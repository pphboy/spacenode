package appaider

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"spacenode/libs/models"
	"spacenode/libs/utils"

	"github.com/sirupsen/logrus"
)

const (
	LzcappDockerComposeDir = "/lzcsys/run/data/system/pkgm/apps"
	LzcappVar              = "/lzcsys/run/data/app/var"
	LzcBinDir              = "/lzcapp/pkg/content"
	DockerComposeFilename  = "docker-compose.yml"
)

type LzcAppHooker interface {
	UpAppPermission(appid string) error
	GenerateConfig(appid string, service string, spc *models.SpaceAppNodeConfig) error
	RunNode(pid int, appid string, service string) (*exec.Cmd, error)
}

type lzcAppHooker struct{}

func NewLzcAppHooker() LzcAppHooker {
	return &lzcAppHooker{}
}

// 提升app docker.compose.yml的权限，支持tun设备的创建
func (h *lzcAppHooker) UpAppPermission(appid string) error {
	dcfl := filepath.Join(LzcappDockerComposeDir, appid, DockerComposeFilename)
	logrus.Infoln("parse docker compose file: ", dcfl)
	dc, err := utils.ParseDockerCompose(dcfl)
	if err != nil {
		return fmt.Errorf("failed to parse docker compose file: %v", err)
	}
	for k, v := range dc.Services {
		hasDev := false
		for _, d := range v.Devices {
			if d["target"] == "/dev/net/tun" {
				hasDev = true
				break
			}
		}
		hasNet := false
		for _, d := range v.CapAdd {
			if d == "NET_ADMIN" {
				hasNet = true
				break
			}
		}
		if !hasDev {
			v.Devices = append(v.Devices, map[string]string{
				"target":      "/dev/net/tun",
				"source":      "/dev/net/tun",
				"permissions": "rwm",
			})
		}

		if !hasNet {
			v.CapAdd = append(v.CapAdd, "NET_ADMIN")
		}
		dc.Services[k] = v
	}

	// 将修改后的文件保存到原来的位置
	if err := utils.SaveDockerCompose(dcfl, dc); err != nil {
		return fmt.Errorf("failed to save docker compose file: %v", err)
	}
	return nil
}

// 生成配置到对应的应用/lzcapp/var下
func (h *lzcAppHooker) GenerateConfig(appid string, service string, spc *models.SpaceAppNodeConfig) error {
	if err := utils.SaveDockerCompose(filepath.Join(LzcappVar, appid, fmt.Sprintf("lzcspace_%s.yml", service)), spc); err != nil {
		return err
	}

	binName := "lzcspacenode"
	// srcBin := filepath.Join(LzcBinDir, binName)
	srcBin := filepath.Join("/lzcapp/var", binName)
	targetDir := filepath.Join(LzcappVar, appid, binName)
	logrus.Infof("Copying %s to %s", srcBin, targetDir)
	if err := utils.CopyFile(srcBin, targetDir); err != nil {
		return err
	}
	return nil
}

// TODO: 如果有必要，可以加上日志，但第1版不加太多功能
// 运行lzcspacenode
func (h *lzcAppHooker) RunNode(pid int, appid string, service string) (*exec.Cmd, error) {
	// 使用nsenter来运行
	// 所以里面的配置只能用auto ip
	// nsenter -t <pid>
	// binPath := filepath.Join(LzcappVar, appid, "lzcspacenode")

	// 执行的位置是容器内
	binPath := filepath.Join("/lzcapp/var", "lzcspacenode")
	cfg := filepath.Join("/lzcapp/var", fmt.Sprintf("lzcspace_%s.yml", service))
	d, err := utils.RunRtrCMD("nsenter", "-n", "-m", "-t", fmt.Sprint(pid), binPath, "-config", cfg)
	logrus.Infoln("AppId ", appid, " DockerPid: ", pid, " NsenterPid: ", d.Process.Pid)
	if err != nil {
		return nil, fmt.Errorf("failed to run node: %v", err)
	}
	return d, nil
}
