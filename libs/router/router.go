package router

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"spacenode/libs/models"
	"spacenode/libs/syncmap"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

type Router struct {
	syncmap.SyncMap[string, *models.RegisterRequest]
	routerMap map[string]net.Conn
}

func NewRouter() *Router {
	return &Router{
		routerMap: make(map[string]net.Conn),
	}
}

func (r *Router) Register(ip string, conn net.Conn) {
	logrus.Info("register ip: ", ip)
	r.routerMap[ip] = conn
}

func (r *Router) Remove(ip string) {
	if conn, ok := r.routerMap[ip]; ok {
		conn.Close()
		delete(r.routerMap, ip)
	}
}

func (r *Router) Serve(ip string) error {
	conn, ok := r.routerMap[ip]
	if !ok {
		return fmt.Errorf("ip %s not found", ip)
	}
	defer conn.Close()

	for {
		lengthBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, lengthBuf); err != nil {
			if err == io.EOF {
				delete(r.routerMap, ip)
				logrus.Infof("connection closed for ip %s", ip)
				return nil
			}
			logrus.Errorf("读取长度头失败: %v", err)
			return err
		}

		pktLength := binary.BigEndian.Uint16(lengthBuf)
		// 读取实际数据
		packetData := make([]byte, pktLength)
		if _, err := io.ReadFull(conn, packetData); err != nil {
			if err == io.EOF {
				delete(r.routerMap, ip)
				logrus.Infof("connection closed for ip %s", ip)
				return nil
			}
			logrus.Errorf("读取数据体失败: %v", err)
			return err
		}

		packet := gopacket.NewPacket(packetData, layers.LayerTypeIPv4, gopacket.Default)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			logrus.Errorln("skip ")
			continue
		}
		ipv4, _ := ipLayer.(*layers.IPv4)

		// 转发逻辑
		targetConn, exist := r.routerMap[ipv4.DstIP.String()]
		if exist {
			lengthBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBuf, uint16(len(packetData)))
			dataToSend := append(lengthBuf, packetData...)
			if _, err := targetConn.Write(dataToSend); err != nil {
				logrus.Errorf("write error: %v", err)
			}
		}
	}
}

func (r *Router) Stop() {
	for ip, conn := range r.routerMap {
		if err := conn.Close(); err != nil {
			logrus.Errorf("failed to close connection for ip %s: %v", ip, err)
		}
		delete(r.routerMap, ip)
	}
}
