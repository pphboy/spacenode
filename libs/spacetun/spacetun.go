package spacetun

import (
	"fmt"
	"os/exec"
	"spacenode/libs/models"
	"spacenode/libs/utils"

	"github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

func SetupTUN(tsc *models.TunSetupConfig) (*water.Interface, error) {
	// 开机的时候尝试关闭一下
	if err := closeTun(tsc); err != nil {
		logrus.Errorln("setup tun, close tun", err)
		return nil, err
	}
	config := water.Config{
		DeviceType: water.TUN,
	}
	config.Name = tsc.Name
	ifce, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	if tsc.IPv4 != "" {
		// 配置IP地址
		cmd := exec.Command("ip", "addr", "add", tsc.IPv4, "dev", ifce.Name())
		if err := cmd.Run(); err != nil {
			ifce.Close()
			return nil, fmt.Errorf("failed to configure IP: %w", err)
		}
	}

	// 启用设备
	cmd := exec.Command("ip", "link", "set", "dev", ifce.Name(), "up")
	if err := cmd.Run(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("failed to bring up interface: %w", err)
	}

	return ifce, nil
}

func closeTun(tsc *models.TunSetupConfig) error {
	config := water.Config{
		DeviceType: water.TUN,
	}
	config.Name = tsc.Name
	ifce, err := water.New(config)
	if err != nil {
		return fmt.Errorf("failed to create TUN device: %w", err)
	}
	defer ifce.Close()
	if _, err := utils.Run("ip", "link", "set", "dev", ifce.Name(), "down"); err != nil {
		return err
	}

	if _, err := utils.Run("ip", "link", "delete", ifce.Name()); err != nil {
		return err
	}

	return nil
}
