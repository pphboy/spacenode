package appaider

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type LzcDockerContainer struct {
	ContainerID string
	Pid         int
	Name        string
}

const (
	LzcDockerSock = "unix:///lzcsys/run/lzc-docker/docker.sock"
)

type LzcDockerHolder interface {
	ListContainers(appid string) ([]LzcDockerContainer, error)
	Close() error
}

type lzcDockerHolder struct {
	dockerCli *client.Client
}

func NewLzcDockerHolder() (LzcDockerHolder, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(LzcDockerSock),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	h := &lzcDockerHolder{
		dockerCli: cli,
	}
	return h, nil
}

func (h *lzcDockerHolder) ListContainers(appid string) ([]LzcDockerContainer, error) {
	filter := filters.NewArgs()
	filter.Add("label", "home-cloud.app-id="+appid)

	containers, err := h.dockerCli.ContainerList(
		context.Background(),
		container.ListOptions{
			All:     true,   // 包含所有容器（运行中+已停止）
			Filters: filter, // 应用标签过滤器[7,9](@ref)
		},
	)
	if err != nil {
		return nil, err
	}
	var ldcs []LzcDockerContainer
	for _, v := range containers {
		detail, err := h.dockerCli.ContainerInspect(context.Background(), v.ID)
		if err != nil {
			logrus.Errorf("检查容器 %s 失败: %v\n", v.ID, err)
			continue
		}

		ldcs = append(ldcs, LzcDockerContainer{
			ContainerID: detail.ID,
			Name:        strings.ReplaceAll(detail.Name, "/", ""),
			Pid:         detail.State.Pid,
		})
	}
	return ldcs, nil
}

func (h *lzcDockerHolder) Close() error {
	return h.dockerCli.Close()
}
