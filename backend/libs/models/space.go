package models

import "time"

type AppStatus string

const (
	AppStatusRunning  AppStatus = "running"
	AppStatusStopped  AppStatus = "stopped"
	AppStatusError    AppStatus = "error"
	AppStatusDisabled AppStatus = "disabled"
)

type NodeType string

// app,client
const (
	NodeTypeApp    NodeType = "app"
	NodeTypeClient NodeType = "client"
)

type AppNode struct {
	SpaceID   string    `json:"space_id" gorm:"primaryKey"`
	NodeID    string    `json:"node_id" gorm:"-"`
	AppID     string    `json:"app_id" gorm:"primaryKey"`
	Status    AppStatus `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ClientNode struct {
	ClientID     string `json:"client_id"`
	DeviceName   string `json:"device_name"`
	DevicePeerID string `json:"device_peer_id"`
	DeviceType   string `json:"device_type"` // win, linux
}

type SpaceItemConfig struct {
	Port    int    `yaml:"port" json:"port"`
	Host    string `yaml:"host" json:"host"`
	ID      string `json:"id" yaml:"id" gorm:"primaryKey"`
	NetAddr string `json:"net_addr" yaml:"net_addr"`
	Mask    string `json:"mask" yaml:"mask"`
}

type SpaceNode struct {
	SpaceID   string   `json:"space_id" yaml:"space_id"`
	NodeID    string   `json:"node_id" yaml:"node_id"`
	NodeType  NodeType `json:"node_type" yaml:"node_type"`
	DockerPid int      `json:"docker_pid" yaml:"docker_pid"`
	AppID     string   `json:"app_id" yaml:"app_id"`
	Service   string   `json:"service" yaml:"service"`
	Domain    string   `json:"domain" yaml:"domain"`
}

type SpaceAppNodeConfig struct {
	NodeConfig  SpaceNode       `json:"node_config" yaml:"node_config"`
	SpaceConfig SpaceItemConfig `json:"space_config" yaml:"space_config"`
}
