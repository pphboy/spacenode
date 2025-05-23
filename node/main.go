package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net"
	"spacenode/libs/models"
	"spacenode/libs/spacetun"
	"time"

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

	logrus.Info("Entering main loop to read from TUN and write to connection")
	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		tcpConn.SetNoDelay(true) // 禁用 Nagle 算法
	}

	go func() {
		buf := make([]byte, 1024) // 32KB 缓冲区
		for {
			n, err := tcpConn.Read(buf)
			if err != nil {
				logrus.Errorf("conn 读取失败: %v", err)
				break
			}
			if _, err := ifce.Write(buf[:n]); err != nil {
				logrus.Errorf("%s", buf[:n])
				logrus.Errorf("ifce 写入失败: %v, %d", err, n)
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024) // 32KB 缓冲区
		for {
			n, err := ifce.Read(buf)
			if err != nil {
				logrus.Errorf("ifce 读取失败: %v", err)
				break
			}
			if _, err := tcpConn.Write(buf[:n]); err != nil {
				logrus.Errorf("conn 写入失败: %v", err)
				break
			}
		}
	}()
	select {}
}
