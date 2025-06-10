package space

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"spacenode/libs/ippool"
	"spacenode/libs/models"
	"spacenode/libs/router"
	"spacenode/libs/syncmap"
	"time"

	"github.com/sirupsen/logrus"
)

type NodeItem struct {
	Node models.SpaceNode `json:"node"`
	IP   string           `json:"ip"`
}

type Space struct {
	config models.SpaceItemConfig
	ipPool *ippool.IPPool
	router *router.Router
	nodes  syncmap.SyncMap[string, *NodeItem]
	close  func()
	ctx    context.Context
}

func NewSpace(config models.SpaceItemConfig) (*Space, error) {
	pl, err := ippool.NewIPPool(config.NetAddr, config.Mask)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm := &Space{
		config: config,
		router: router.NewRouter(),
		ipPool: pl,
		ctx:    ctx,
		close:  cancel,
	}
	return sm, nil
}

func (s *Space) Remove(r models.SpaceNode) error {
	logrus.Infof(" remove node %s from space", r.NodeID)
	ni, ok := s.nodes.Load(r.NodeID)
	if !ok {
		return fmt.Errorf("node %s not found", r.NodeID)
	}
	s.router.Remove(ni.IP)
	s.nodes.Delete(r.NodeID)
	return nil
}

// condition: {offline, online, all}
func (s *Space) Nodelist() []*NodeItem {
	arr := make([]*NodeItem, 0)
	s.nodes.Range(func(key string, value *NodeItem) bool {
		arr = append(arr, value)
		return true
	})
	return arr
}

func (s *Space) GetConifg() *models.SpaceItemConfig {
	return &s.config
}

func (s *Space) SetConifg(sc models.SpaceItemConfig) {
	// unimplemented
	logrus.Fatalln("set config not implemented")
}

func (s *Space) Start() error {
	logrus.Infoln("space manager start", "host:", s.config.Host, "port:", s.config.Port, "id:", s.config.ID)
	lis, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.config.Host, s.config.Port))
	if err != nil {
		return err
	}
	defer lis.Close()

	for {
		select {
		case <-s.ctx.Done():
			logrus.Warnln("space manager stopped")
			return nil
		default:
		}
		conn, err := lis.Accept()
		if err != nil {
			logrus.Errorln("accpet", err)
			continue
		}

		go func(conn net.Conn) {
			// 1. 读配置
			req := &models.RegisterRequest{}
			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)
			line, err := reader.ReadBytes('\n')
			if err != nil {
				logrus.Errorln("read", err)
				return
			}
			if err := json.NewDecoder(bytes.NewReader(line)).Decode(req); err != nil {
				logrus.Errorln("json decode", err)
				return
			}
			// 2. 处理客户端ip
			ip, err := s.AssignIP(req)
			if err != nil {
				logrus.Errorln("assign ip", err)
				return
			}
			respBf := bytes.NewBuffer(nil)
			resp := &models.RegisterResp{
				IPv4:  ip,
				Alive: 30 * 24 * time.Hour,
			}
			if err := json.NewEncoder(respBf).Encode(resp); err != nil {
				logrus.Errorln("json encode", err)
				return
			}
			respBf.WriteByte('\n')
			_, err = conn.Write(respBf.Bytes())
			if err != nil {
				logrus.Errorln("write", err)
				return
			}
			// 4. 注册链接
			s.router.Register(resp.IPv4, conn)
			defer s.router.Remove(resp.IPv4)
			// TODO: 心跳检测
			// 3. 写回执
			writer.Write(respBf.Bytes())
			writer.WriteByte('\n')

			s.nodes.Store(req.SpaceNode.NodeID, &NodeItem{
				Node: req.SpaceNode,
				IP:   resp.IPv4,
			})
			// 6: 路由
			if err := s.router.Serve(resp.IPv4); err != nil {
				logrus.Errorln("s router serve", req, " ", err)
				return
			}
		}(conn)
	}

}

func (s *Space) AssignIP(req *models.RegisterRequest) (string, error) {
	if req.NetConfig.DHCPType == "auto" {
		// 分配随机IP
		ip, err := s.ipPool.Random(24 * 30 * time.Hour)
		if err != nil {
			return "", err
		}
		return ip.String(), nil
	} else if req.NetConfig.DHCPType == "static" {
		// 分配随机IP
		bl, err := s.ipPool.RequestIP(req.NetConfig.IPv4, 24*30*time.Hour)
		if err != nil {
			return "", err
		}
		if !bl {
			return "", fmt.Errorf("ip %s is already used", req.NetConfig.IPv4)
		}
		return req.NetConfig.IPv4, nil
	}
	return "", nil
}

func (s *Space) Stop() error {
	s.router.Stop()
	s.close()
	logrus.Infof("%s: server stopped", s.config.ID)
	return nil
}
