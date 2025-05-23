package router

import (
	"fmt"
	"net"
	"spacenode/libs/models"
	"spacenode/libs/syncmap"
	"time"

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

	bf := make([]byte, 65535) // IPv4最大长度（64KB）
	for {
		// 读取数据
		n, err := conn.Read(bf)
		if err != nil {
			logrus.Errorln(time.Now().Second())
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // 超时重试
			}
			logrus.Errorf("read error: %v", err)
			return err
		}
		origin := make([]byte, n)
		copy(origin, bf[:n])

		// 解析IP包
		packet := gopacket.NewPacket(origin, layers.LayerTypeIPv4, gopacket.Default)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			continue
		}
		ipv4, _ := ipLayer.(*layers.IPv4)

		// 转发逻辑
		targetConn, exist := r.routerMap[ipv4.DstIP.String()]
		if exist {
			if _, err := targetConn.Write(bf[:n]); err != nil {
				logrus.Errorf("write error: %v", err)
			}
		}
	}
}
