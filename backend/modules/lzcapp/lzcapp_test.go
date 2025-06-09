package lzcapp

import (
	"context"
	"spacenode/libs/lzcutils"
	"testing"
)

func TestAppList(t *testing.T) {
	manager, err := NewLzcAppManager()
	if err != nil {
		t.Fatalf("Failed to create LzcAppManager: %v", err)
	}

	ctx := context.Background()
	apps, err := manager.AppList(lzcutils.ToGrpcContext(ctx, "dzh"))
	if err != nil {
		t.Fatalf("AppList failed: %v", err)
	}

	t.Logf("Got %d apps", len(apps))
	for _, app := range apps {
		t.Logf("App ID: %s, Name: %s", app.Appid, *app.Domain)
	}
}

func TestRestartApp(t *testing.T) {
	manager, err := NewLzcAppManager()
	if err != nil {
		t.Fatalf("Failed to create LzcAppManager: %v", err)
	}

	ctx := context.Background()
	testAppID := "cloud.lazycat.app.fiai"
	err = manager.RestartApp(lzcutils.ToGrpcContext(ctx, "dzh"), testAppID)
	if err != nil {
		t.Fatalf("RestartApp failed: %v", err)
	}

	t.Logf("Successfully restarted app %s", testAppID)
}
