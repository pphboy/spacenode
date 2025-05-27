package main

import (
	"flag"
	"os"
	"spacenode/libs/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	cfgPath = flag.String("config", "/lzcapp/var/config.yml", "Path to the configuration file")
)

func main() {
	flag.Parse()
	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		logrus.Warn("No configuration file provided or failed to load config, using default configuration")
		cfg = &config.Config{
			Name: "test",
			Addr: "172.168.1.1",
			Mask: "255.255.255.0",
			Port: 59393,
			Host: "host.lzcapp",
		}
		logrus.Warnf("Using default configuration: %+v", cfg)
	}
	logrus.SetReportCaller(true)
	s, err := NewServer(cfg)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := s.Start(); err != nil {
			panic(err)
		}
	}()
	e := gin.Default()

	e.GET("/", func(ctx *gin.Context) {
		ctx.String(200, "Hello, LzcSpace!")
	})

	if err := e.Run(":8080"); err != nil {
		logrus.Warnln(err)
	}
}

// LoadConfig 加载配置文件，返回 *config.Config 或 (nil, error)
func LoadConfig(path string) (*config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg config.Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
