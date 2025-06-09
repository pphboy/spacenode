package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"runtime"
	"spacenode/libs/models"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

var (
	moon = flag.String("ipaddr", "172.23.253.179:9393", "MoonServer IP")
)

func main() {
	flag.Parse()
	logrus.SetReportCaller(true)

	logrus.Info("Creating register request")
	rr := &models.RegisterRequest{
		NodeName: "node" + uuid.New().String()[:3],
		NetConfig: models.NetConfig{
			Type:     "ipv4",
			DHCPType: "auto",
			Alive:    time.Hour * 30 * 24,
		},
		MoonServer: *moon,
	}

	logrus.Infof("Dialing MoonServer at %s", rr.MoonServer)
	conn, err := net.Dial("tcp", rr.MoonServer)
	if err != nil {
		logrus.Fatalf("Failed to dial MoonServer: %v", err)
	}
	defer conn.Close()

	logrus.Info("Encoding register request to JSON")
	requestBF := &bytes.Buffer{}
	if err := json.NewEncoder(requestBF).Encode(rr); err != nil {
		logrus.Fatalf("Failed to encode register request: %v", err)
	}
	requestBF.WriteByte('\n')

	logrus.Info("Sending register request")
	_, err = conn.Write(requestBF.Bytes())
	if err != nil {
		logrus.Fatalf("Failed to send register request: %v", err)
	}

	packet := make([]byte, 1024)
	logrus.Info("Waiting for response from MoonServer")
	n, err := conn.Read(packet)
	if err != nil {
		logrus.Fatalf("Failed to read response: %v", err)
	}

	response := &models.RegisterResp{}
	logrus.Info("Decoding response from MoonServer")
	if err := json.Unmarshal(packet[:n], response); err != nil {
		logrus.Fatalf("Failed to decode response: %v", err)
	}
	logrus.Info("Response from MoonServer received", response)

	logrus.Info("Setting up TUN interface")
	ifce, err := createWindowsTunl(response)
	if err != nil {
		logrus.Fatalf("Failed to setup TUN interface: %v", err)
		return
	}
	defer ifce.Close()
	defer conn.Close()
	logrus.Infoln("batchsize", ifce.BatchSize())

	go func() {
		for {
			lengthBuf := make([]byte, 2)
			if _, err := io.ReadFull(conn, lengthBuf); err != nil {
				if err == io.EOF {
					logrus.Errorln("MoonServer 关闭连接")
					return
				}
				logrus.Errorf("读取长度头失败: %v", err)
				continue
			}

			pktLength := binary.BigEndian.Uint16(lengthBuf)
			// 读取实际数据
			packetData := make([]byte, pktLength)
			if _, err := io.ReadFull(conn, packetData); err != nil {
				logrus.Errorf("读取数据体失败: %v", err)
				continue
			}
			packet := gopacket.NewPacket(packetData, layers.LayerTypeIPv4, gopacket.Default)
			ipLayer := packet.Layer(layers.LayerTypeIPv4)
			if ipLayer == nil {
				logrus.Errorln("skip ")
				continue
			}
			ipv4, _ := ipLayer.(*layers.IPv4)

			logrus.Infoln(ipLayer.LayerType(), ipv4.SrcIP, ipv4.DstIP, "len", pktLength)
			if _, err := ifce.Write([][]byte{packetData}, 0); err != nil {
				logrus.Errorf("ifce 写入失败: %v", err)
				continue
			}
		}
	}()

	go func() {
		bufs := make([][]byte, ifce.BatchSize())
		sizes := make([]int, ifce.BatchSize())
		for i := range bufs {
			bufs[i] = make([]byte, 1500) // MTU大小
		}
		for {
			n, err := ifce.Read(bufs, sizes, 0)
			if err != nil {
				logrus.Errorf("ifce 读取失败: %v", err)
				break
			}
			lengthBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBuf, uint16(sizes[0]))

			logrus.Infoln("readpages", n, "sizes", sizes)

			dataToSend := append(lengthBuf, bufs[0][:sizes[0]]...)

			if _, err := conn.Write(dataToSend); err != nil {
				logrus.Errorf("conn 写入失败: %v", err)
				break
			}
		}
	}()
	select {}
}

func createWindowsTunl(resp *models.RegisterResp) (tun.Device, error) {
	// 设置TUN设备名称（Windows需遵循命名规范）
	devName := "MyTun@" + uuid.NewString()[:3]
	if runtime.GOOS == "windows" {
		devName = "Wintun" + uuid.NewString()[:3]
	}

	// 创建TUN设备（参数说明：设备名, MTU大小）
	wintun, err := tun.CreateTUN(devName, 1500)
	if err != nil {
		logrus.Errorf("创建TUN设备失败: %v", err)
		return nil, err
	}
	// 获取实际设备名称（不同系统可能自动重命名）
	actualName, err := wintun.Name()
	if err != nil {
		logrus.Errorf("获取设备名称失败: %v", err)
		return nil, err
	}
	logrus.Printf("已创建TUN设备: %s", actualName)
	configureTUN(wintun, resp)
	return wintun, nil
}

func configureTUN(dev tun.Device, resp *models.RegisterResp) {
	nativeTun := dev.(*tun.NativeTun)
	luid := winipcfg.LUID(nativeTun.LUID())

	// 设置IP地址（例如10.0.0.2/24）
	ip, _ := netip.ParsePrefix(fmt.Sprintf("%s/24", resp.IPv4))
	err := luid.SetIPAddresses([]netip.Prefix{ip})
	if err != nil {
		logrus.Fatal("Failed to set IP:", err)
	}

	// // 添加路由（例如将所有流量路由到TUN设备）
	// route := &winipcfg.RouteData{
	// 	Destination: netip.MustParsePrefix("0.0.0.0/0"),
	// 	NextHop:     netip.MustParseAddr("10.0.0.1"), // 服务器端TUN设备IP
	// 	Metric:      0,
	// }

	// err = luid.SetRoutes([]*winipcfg.RouteData{route})
	// if err != nil {
	// 	log.Fatal("Failed to set route:", err)
	// }
}
