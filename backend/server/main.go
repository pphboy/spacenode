package main

import (
	"flag"
	"os"
	"os/signal"
	"spacenode/libs/models"
	"spacenode/modules/db"
	"spacenode/modules/space"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	dbpath = flag.String(
		"dbpath",
		"/lzcapp/var/db.db",
		"database path",
	)
)

func main() {
	flag.Parse()
	logrus.SetReportCaller(true)
	db.InitDB(*dbpath)

	sm, err := space.NewSpace(models.SpaceItemConfig{
		Port:    59393,
		Host:    "host.lzcapp",
		ID:      "space1",
		NetAddr: "172.168.1.0",
		Mask:    "255.255.255.0",
	})
	if err != nil {
		logrus.Fatalln("failed to create space manager: ", err)
	}

	go func() {
		if err := sm.Start(); err != nil {
			logrus.Warnln("space manager stopped: ", err)
		}
	}()

	// 设置信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或上下文取消
	select {
	case sig := <-sigChan:
		sm.Stop()
		logrus.Infof("received signal %v, shutting down gracefully", sig)
	}
}
