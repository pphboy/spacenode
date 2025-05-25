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

// TODO: 1. 要处理好断联的操作
func (r *Router) Serve2(ip string) error {
	conn, ok := r.routerMap[ip]
	if !ok {
		return fmt.Errorf("ip %s not found", ip)
	}
	bf := make([]byte, 65535)
	for {
		n, err := conn.Read(bf)
		if err != nil {
			logrus.Errorf("read error: %v", err)
			return err
		}
		if n == 0 {
			continue
		}
		packet := gopacket.NewPacket(bf[:n], layers.LayerTypeIPv4, gopacket.Default)
		ipL := packet.Layer(layers.LayerTypeIPv4)
		if ipL != nil {
			ip, _ := ipL.(*layers.IPv4)
			logrus.Infoln(ip.SrcIP, ip.DstIP, ip.Protocol)
			targetConn, exist := r.routerMap[ip.DstIP.String()]
			if exist {
				if _, err := targetConn.Write(bf[:n]); err != nil {
					logrus.Errorf("write error: %v", err)
				}
				logrus.Infoln("write ok")
			}
		}
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
				return nil
			}
			logrus.Errorf("读取长度头失败: %v", err)
			return err
		}

		pktLength := binary.BigEndian.Uint16(lengthBuf)
		// 读取实际数据
		packetData := make([]byte, pktLength)
		if _, err := io.ReadFull(conn, packetData); err != nil {
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
