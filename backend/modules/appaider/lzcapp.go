package appaider

import (
	"fmt"
	"os"
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
	DockerComposeFilename  = "compose.override.yml"
	Manifest               = "manifest.yml"
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
	// 读manifest.yml中的 services
	mf := filepath.Join(LzcappDockerComposeDir, appid, "pkg", Manifest)
	if !utils.FileExists(mf) {
		return fmt.Errorf("appid %s manifest file not found: %s ", appid, mf)
	}

	var err error
	mc, err := utils.ParseManifest(mf)
	if err != nil {
		return fmt.Errorf("failed to parse manifest %s %v", mf, err)
	}

	if mc.Services == nil {
		mc.Services = make(map[string]utils.MService)
	}

	// 默认将app添加进去
	mc.Services["app"] = utils.MService{}
	var dc *utils.DockerCompose

	dcfl := filepath.Join(LzcappDockerComposeDir, appid, "pkg", DockerComposeFilename)
	if !utils.FileExists(dcfl) {
		dc = &utils.DockerCompose{}
	} else {
		logrus.Infoln("parse docker compose file: ", dcfl)
		dc, err = utils.ParseDockerCompose(dcfl)
		if err != nil {
			return fmt.Errorf("failed to parse docker compose file: %v", err)
		}
	}
	if dc.Services == nil {
		dc.Services = make(map[string]utils.Service)
	}

	// 如果不存在，需要补全
	for k := range mc.Services {
		if _, ok := dc.Services[k]; !ok {
			dc.Services[k] = utils.Service{}
		}
	}

	// 将services判断是否已在compose.override.yml中配置
	// 未配置则添加，已配置则需要判断是否已添加/dev/net/tun
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
		// v.Privileged= true
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
	binDir := os.Getenv("LZCSPACENODE_BIN_DIR")
	if binDir == "" {
		return fmt.Errorf("LZCSPACENODE_BIN_DIR is not set")
	}
	// srcBin := filepath.Join(LzcBinDir, binName)
	srcBin := filepath.Join(binDir, binName)

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
	if _, err := utils.RunRtrCMD("nsenter", "-n", "-m", "-t", fmt.Sprint(pid), "chmod", "+x", binPath); err != nil {
		return nil, fmt.Errorf("failed to chmod: %v", err)
	}
	cfg := filepath.Join("/lzcapp/var", fmt.Sprintf("lzcspace_%s.yml", service))
	d, err := utils.RunRtrCMD("nsenter", "-n", "-m", "-t", fmt.Sprint(pid), binPath, "-config", cfg)
	logrus.Infoln("AppId ", appid, " DockerPid: ", pid, " NsenterPid: ", d.Process.Pid)
	if err != nil {
		return nil, fmt.Errorf("failed to run node: %v", err)
	}
	return d, nil
}
