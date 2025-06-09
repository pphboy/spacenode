package appaider

import (
	"spacenode/libs/models"
	"testing"
)

func TestLzcAppHooker_UpAppPermission(t *testing.T) {
}

func TestLzcAppHooker_GenerateConfig(t *testing.T) {
}

func TestLzcAppHooker_RunNode(t *testing.T) {
	lah := NewLzcAppHooker()
	if err := lah.UpAppPermission("cloud.lazycat.app.fiai"); err != nil {
		t.Fatalf("UpApp Permission failed: %v", err)
	}
	//
	if err := lah.GenerateConfig("cloud.lazycat.app.fiai", "app", &models.SpaceAppNodeConfig{
		SpaceConfig: models.SpaceItemConfig{
			Port:    59393,
			Host:    "host.lzcapp",
			ID:      "test-id",
			NetAddr: "172.168.1.100",
			Mask:    "255.255.255.0",
		},
		NodeConfig: models.SpaceNode{
			SpaceID:  "test-space-id",
			NodeID:   "test-node-id",
			NodeType: "test-node-type",
		}}); err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}
	// WARN: 还缺的操作，需要修改之后将应用重启
	// 在docker容器中运行
	if _, err := lah.RunNode(1851688, "cloud.lazycat.app.fiai", "app"); err != nil {
		t.Fatalf("RunNode failed: %v", err)
	}
}
