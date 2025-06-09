package appaider

import (
	"context"
	"fmt"
	"os/exec"
	"spacenode/libs/models"
	"spacenode/libs/syncmap"
	"spacenode/modules/lzcapp"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AppAider interface {
	List() []*models.AppNode
	Add(ctx context.Context, a *models.AppNode) error    // 修改为指针类型
	Remove(ctx context.Context, a *models.AppNode) error // 修改为指针类型
	// TODO: 未来的功能，应由未来实现
}

type appAider struct {
	db        *gorm.DB
	apps      syncmap.SyncMap[string, *models.AppNode]
	appTop    syncmap.SyncMap[string, *syncmap.SyncMap[string, *exec.Cmd]]
	lam       lzcapp.LzcAppManager
	hooker    LzcAppHooker
	lzcdocker LzcDockerHolder
}

// 实现AppAider
func NewAppAider(db *gorm.DB, lzcAppManager lzcapp.LzcAppManager) (AppAider, error) {
	holder, err := NewLzcDockerHolder()
	if err != nil {
		return nil, err
	}
	ai := &appAider{
		db:        db,
		lam:       lzcAppManager,
		hooker:    NewLzcAppHooker(),
		lzcdocker: holder,
	}
	return ai, nil
}

func (a *appAider) List() []*models.AppNode {
	arr := make([]*models.AppNode, 0)
	a.apps.Range(func(key string, value *models.AppNode) bool {
		arr = append(arr, value)
		return true
	})
	return arr
}

// nsenter的进程控制权限，在lzcspace下
func (a *appAider) Add(ctx context.Context, an *models.AppNode) error {
	if _, ok := a.apps.Load(an.AppID); ok {
		return fmt.Errorf("app %s already exists", an.AppID)
	}

	a.appTop.Store(an.AppID, &syncmap.SyncMap[string, *exec.Cmd]{})

	if err := a.hooker.UpAppPermission(an.AppID); err != nil {
		logrus.Errorln("failed to up app permission: ", err)
		return err
	}

	if err := a.lam.RestartApp(ctx, an.AppID); err != nil {
		return err
	}

	logrus.Infof("add app node: %v", an)
	dks, err := a.lzcdocker.ListContainers(an.AppID)
	if err != nil {
		return err
	}

	if len(dks) == 0 {
		logrus.Errorln("no container found for app ", an.AppID)
		return fmt.Errorf("no container found for app %s", an.AppID)
	}

	for _, container := range dks {
		ak := appKey(container.Name, an.AppID)
		a.hooker.GenerateConfig(an.AppID, container.Name, &models.SpaceAppNodeConfig{
			NodeConfig: models.SpaceNode{
				SpaceID:   an.SpaceID,
				NodeID:    an.NodeID,
				NodeType:  "app", // 类型
				DockerPid: container.Pid,
				AppID:     an.AppID,
				Service:   container.Name,
			},
			SpaceConfig: models.SpaceItemConfig{
				Port:    59393, // FIXME: 待后面优化的时候将这个写死的端口去掉
				ID:      ak,
				Host:    "host.lzcapp",
				Mask:    "255.255.255.0",
				NetAddr: "172.168.1.0",
			},
		})
		c, err := a.hooker.RunNode(container.Pid, an.AppID, container.Name)
		if err != nil {
			logrus.Errorln("failed to run node: ", err)
			continue
		}
		// 将nsenter的进程控制存储起来
		if top, ok := a.appTop.Load(an.AppID); ok {
			top.Store(ak, c)
		}
	}

	a.apps.Store(an.AppID, an)
	if err := a.db.Save(an).Error; err != nil {
		return err
	}
	return nil
}

func (a *appAider) Remove(ctx context.Context, an *models.AppNode) error {
	logrus.Infof("remove app: %s", an.AppID)
	if top, ok := a.appTop.Load(an.AppID); ok {
		top.Range(func(key string, value *exec.Cmd) bool {
			logrus.Infof("try to kill nsenter %s, Pid: %d", key, value.Process.Pid)
			if err := value.Process.Kill(); err != nil {
				logrus.Errorln("failed to kill nsenter: ", err)
			}
			top.Delete(key)
			return true
		})
		a.appTop.Delete(an.AppID)
	}

	a.apps.Delete(an.AppID)
	if err := a.db.Delete(an).Error; err != nil {
		return err
	}
	return nil
}

func appKey(containerName string, appID string) string {
	return fmt.Sprintf("%s.%s.lzcapp", containerName, appID)
}
