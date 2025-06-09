package router

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"spacenode/libs/syncmap"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

type routerItem struct {
	IP     string
	cancel func()
	ctx    context.Context
}

type Router struct {
	routerMap syncmap.SyncMap[string, net.Conn]
	items     syncmap.SyncMap[string, *routerItem]
}

func NewRouter() *Router {
	r := Router{}
	return &r
}

func (r *Router) Register(ip string, conn net.Conn) {
	logrus.Info("register ip: ", ip)
	r.routerMap.Store(ip, conn)
	ctx, cancel := context.WithCancel(context.Background())
	r.items.Store(ip, &routerItem{
		IP:     ip,
		cancel: cancel,
		ctx:    ctx,
	})

}

func (r *Router) Remove(ip string) {
	if conn, ok := r.routerMap.Load(ip); ok {
		if err := conn.Close(); err != nil {
			logrus.Warnf("close conn error: %v", err)
		}
		r.routerMap.Delete(ip)
	}
	item, ok := r.items.Load(ip)
	if ok {
		item.cancel()
		r.items.Delete(ip)
	}

}

func (r *Router) Serve(ip string) error {
	item, ok := r.items.Load(ip)
	if !ok {
		return fmt.Errorf("ip %s not found", ip)
	}
	conn, ok := r.routerMap.Load(ip)
	if !ok {
		return fmt.Errorf("ip %s not found", ip)
	}
	defer conn.Close()

	for {
		select {
		case <-item.ctx.Done():
			logrus.Infof("ip %s router closed", ip)
			return nil
		default:
		}
		lengthBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, lengthBuf); err != nil {
			if err == io.EOF {
				r.routerMap.Delete(ip)
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
				r.routerMap.Delete(ip)
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
		targetConn, exist := r.routerMap.Load(ipv4.DstIP.String())
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
	r.routerMap.Range(func(ip string, conn net.Conn) bool {
		if err := conn.Close(); err != nil {
			logrus.Errorf("failed to close connection for ip %s: %v", ip, err)
		}
		r.routerMap.Delete(ip)
		return true
	})
}
