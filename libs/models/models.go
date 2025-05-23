package models

import "time"

type RegisterRequest struct {
	MoonServer string    `yaml:"moon_server" json:"moon_server"` // moon server的服务地址
	NodeName   string    `yaml:"node_name" json:"node_name"`     // 客户端名称
	NetConfig  NetConfig `yaml:"net_config" json:"net_config"`   //注册请求的配置
}

// 请求
type NetConfig struct {
	Type     string `yaml:"type" json:"type"`       // ipv4,ipv6,ip
	DHCPType string `yaml:"dhcptyp" json:"dhcptyp"` //auto,static
	// 仅static有效
	IPv4 string `yaml:"addr" json:"addr"`
	// Alive seconds
	Alive time.Duration `yaml:"alive" json:"alive"`
}

// 申请的返回结果
type RegisterResp struct {
	// 仅static有效
	IPv4 string `yaml:"addr" json:"addr"`

	Alive time.Duration `yaml:"alive" json:"alive"`
}

// 给tun_setup使用的
type TunSetupConfig struct {
	IPv4 string `json:"ipv4"`
	Name string `json:"name"`
}
