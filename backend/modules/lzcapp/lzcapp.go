package lzcapp

import (
	"context"
	"fmt"

	ll "gitee.com/linakesi/lzc-sdk/lang/go"
	"gitee.com/linakesi/lzc-sdk/lang/go/sys"
	"google.golang.org/grpc"
)

type LzcAppManager interface {
	AppList(ctx context.Context) ([]*sys.AppInfo, error)
	RestartApp(ctx context.Context, appid string) error
}

const (
	lzcBoxSock = ""
)

func NewLzcAppManager() (LzcAppManager, error) {
	fmt.Printf("try init lazy runtime conn\n")
	wrapErr := func(err error) error { return fmt.Errorf("lazy runtime conn init failed, err = %+v", err) }
	cred, err := ll.BuildClientCredOption(
		"/lzcsys/var/runtime/config/apicerts/box.crt",
		"/lzcsys/var/runtime/config/apicerts/app.key",
		"/lzcsys/var/runtime/config/apicerts/app.crt",
	)
	if err != nil {
		return nil, wrapErr(err)
	}
	conn, err := grpc.NewClient("unix:///lzcapp/run/sys//lzc-apis.socket", cred) // 有 cred 加载的操作，可以保证 runtime 未启动时此处立刻报错，无需额外检测
	if err != nil {
		return nil, wrapErr(err)
	}
	return &lzcAppManager{
		client: sys.NewPackageManagerClient(conn),
	}, nil
}

type lzcAppManager struct {
	client sys.PackageManagerClient
}

func (m *lzcAppManager) AppList(ctx context.Context) ([]*sys.AppInfo, error) {
	// ctx需要传一个管理员
	resp, err := m.client.QueryApplication(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.InfoList, nil
}

func (m *lzcAppManager) RestartApp(ctx context.Context, appid string) error {
	// 暂停
	if _, err := m.client.Pause(context.TODO(), &sys.AppInstance{
		Appid: appid,
	}); err != nil {
		return fmt.Errorf(" pause app failed:  ,%w", err)
	}
	// 恢复
	if _, err := m.client.Resume(context.TODO(), &sys.AppInstance{
		Appid: appid,
	}); err != nil {
		return fmt.Errorf(" resume app failed:  ,%w", err)
	}
	return nil
}
