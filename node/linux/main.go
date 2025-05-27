package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"net"
	"spacenode/libs/models"
	"spacenode/libs/spacetun"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	ifce, err := spacetun.SetupTUN(&models.TunSetupConfig{
		Name: rr.NodeName,
		IPv4: response.IPv4 + "/24",
	})
	if err != nil {
		logrus.Fatalf("Failed to setup TUN interface: %v", err)
		return
	}
	defer ifce.Close()
	defer conn.Close()

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

			ifce.Write(packetData)
		}
	}()

	go func() {
		buf := make([]byte, 65535) // 32KB 缓冲区
		for {
			n, err := ifce.Read(buf)
			if err != nil {
				logrus.Errorf("ifce 读取失败: %v", err)
				break
			}
			lengthBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBuf, uint16(n))

			dataToSend := append(lengthBuf, buf[:n]...)

			if _, err := conn.Write(dataToSend); err != nil {
				logrus.Errorf("conn 写入失败: %v", err)
				break
			}
		}
	}()
	select {}
}
