package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"spacenode/libs/config"
	"spacenode/libs/ippool"
	"spacenode/libs/models"
	"spacenode/libs/router"
	"time"

	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg    *config.Config
	ipPool *ippool.IPPool
	router *router.Router
}

func NewServer(cfg *config.Config) (*Server, error) {
	pl, err := ippool.NewIPPool(cfg.Addr, cfg.Mask)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:    cfg,
		ipPool: pl,
		router: router.NewRouter(),
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port))
	if err != nil {
		return err
	}

	for {
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

			// 6: 路由
			if err := s.router.Serve(resp.IPv4); err != nil {
				logrus.Errorln("s router serve", req, " ", err)
				return
			}
		}(conn)
	}
}

func (s *Server) AssignIP(req *models.RegisterRequest) (string, error) {
	if req.NetConfig.DHCPType == "auto" {
		// 分配随机IP
		ip, err := s.ipPool.Random(24 * 30 * time.Hour)
		if err != nil {
			return "", err
		}
		return ip.String(), nil
	} else if req.NetConfig.DHCPType == "static" {
		// 分配随机IP
		bool, err := s.ipPool.RequestIP(req.NetConfig.IPv4, 24*30*time.Hour)
		if err != nil {
			return "", err
		}
		if bool == false {
			return "", fmt.Errorf("ip %s is already used", req.NetConfig.IPv4)
		}
		return req.NetConfig.IPv4, nil
	}
	return "", nil
}

func (s *Server) Stop() error {
	s.router.Stop()
	logrus.Infof("%s: server stopped", s.cfg.Name)
	return nil
}
