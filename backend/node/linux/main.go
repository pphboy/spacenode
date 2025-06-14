package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"spacenode/libs/models"
	"spacenode/libs/spacetun"
	"spacenode/libs/ymlutils"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	moon   = flag.String("ipaddr", "172.23.253.179:9393", "MoonServer IP")
	config = flag.String("config", "", "config file path")
)

// 编译的时候， app / client
var BuildNodeType string

func main() {
	flag.Parse()
	logrus.SetReportCaller(true)

	var log *logrus.Entry
	var rr *models.RegisterRequest
	if *config != "" {
		cfg, err := ymlutils.ParseYAML[*models.SpaceAppNodeConfig](*config)
		if err != nil {
			log.Fatalln(err)
		}
		cfg.NodeConfig.NodeType = models.NodeType(BuildNodeType)
		rr = &models.RegisterRequest{
			SpaceNode: cfg.NodeConfig,
			NodeName:  cfg.NodeConfig.NodeID,
			NetConfig: models.NetConfig{
				Type:     "ipv4",
				DHCPType: "auto",
				Alive:    -1,
			},
			MoonServer: fmt.Sprintf("%s:%d", cfg.SpaceConfig.Host, cfg.SpaceConfig.Port),
		}

		log = logrus.WithField("service", cfg.NodeConfig.Service).
			WithField("appid", rr.SpaceNode.AppID).
			WithField("pid", cfg.NodeConfig.DockerPid)
	} else {
		rr = &models.RegisterRequest{
			SpaceNode: models.SpaceNode{
				NodeID:   "linuxnode_" + uuid.New().String()[:8],
				NodeType: models.NodeType(BuildNodeType),
			},
			NetConfig: models.NetConfig{
				Type:     "ipv4",
				DHCPType: "auto",
				Alive:    time.Hour * 30 * 24,
			},
			MoonServer: *moon,
		}
		log = logrus.WithField("nodeid", rr.SpaceNode.NodeID)
	}
	if BuildNodeType == "client" {
		rr.SpaceNode = models.SpaceNode{
			NodeID:   "linuxnode_" + uuid.New().String()[:8],
			NodeType: models.NodeType(BuildNodeType),
		}
	}

	log.Info("Creating register request")

	log.Infof("Dialing MoonServer at %s", rr.MoonServer)
	conn, err := net.Dial("tcp", rr.MoonServer)
	if err != nil {
		log.Fatalf("Failed to dial MoonServer: %v", err)
	}
	defer conn.Close()

	log.Info("Encoding register request to JSON")
	requestBF := &bytes.Buffer{}
	if err := json.NewEncoder(requestBF).Encode(rr); err != nil {
		log.Fatalf("Failed to encode register request: %v", err)
	}
	requestBF.WriteByte('\n')

	log.Info("Sending register request")
	_, err = conn.Write(requestBF.Bytes())
	if err != nil {
		log.Fatalf("Failed to send register request: %v", err)
	}

	packet := make([]byte, 1024)
	log.Info("Waiting for response from MoonServer")
	n, err := conn.Read(packet)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	response := &models.RegisterResp{}
	log.Info("Decoding response from MoonServer")
	if err := json.Unmarshal(packet[:n], response); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}
	log.Info("Response from MoonServer received", response)

	log.Info("Setting up TUN interface")
	ifce, err := spacetun.SetupTUN(&models.TunSetupConfig{
		Name: rr.NodeName,
		IPv4: response.IPv4 + "/24",
	})
	if err != nil {
		log.Fatalf("Failed to setup TUN interface: %v", err)
		return
	}
	defer ifce.Close()
	defer conn.Close()

	go func() {
		for {
			lengthBuf := make([]byte, 2)
			if _, err := io.ReadFull(conn, lengthBuf); err != nil {
				if err == io.EOF {
					log.Errorln("MoonServer 关闭连接")
					return
				}
				log.Errorf("读取长度头失败: %v", err)
				continue
			}

			pktLength := binary.BigEndian.Uint16(lengthBuf)
			// 读取实际数据
			packetData := make([]byte, pktLength)
			if _, err := io.ReadFull(conn, packetData); err != nil {
				log.Errorf("读取数据体失败: %v", err)
				continue
			}
			packet := gopacket.NewPacket(packetData, layers.LayerTypeIPv4, gopacket.Default)
			ipLayer := packet.Layer(layers.LayerTypeIPv4)
			if ipLayer == nil {
				log.Errorln("skip ")
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
				log.Errorf("ifce 读取失败: %v", err)
				break
			}
			lengthBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBuf, uint16(n))

			dataToSend := append(lengthBuf, buf[:n]...)

			if _, err := conn.Write(dataToSend); err != nil {
				log.Errorf("conn 写入失败: %v", err)
				break
			}
		}
	}()
	select {}
}
