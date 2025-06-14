package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func Run(cmd string, args ...string) (string, error) {
	cs := exec.Command(cmd, args...)
	bf := &bytes.Buffer{}
	cs.Stdout = bf
	cs.Stderr = bf
	if err := cs.Run(); err != nil {
		return "", fmt.Errorf("out: %s, %v", bf.String(), err)
	}
	return bf.String(), nil
}

func RunRtrCMD(cmd string, args ...string) (*exec.Cmd, error) {
	cs := exec.Command(cmd, args...)
	logrus.Debug("RunRtrCMD: ", cs.String())
	cs.Stdout = os.Stdout
	cs.Stderr = os.Stderr
	if err := cs.Start(); err != nil {
		logrus.Error("RunRtrCMD: ", cs.String(), " err:", err)
		return nil, err
	}
	return cs, nil
}

type Manifest struct {
	Services map[string]MService `yaml:"services"`
}

type MService struct {
}

type DockerCompose struct {
	// Version  string                 `yaml:"version"`
	Services map[string]Service `yaml:"services"`
	// Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
	// Networks map[string]interface{} `yaml:"networks,omitempty"`
}

type Service struct {
	// Privileged  bool                `yaml:"privileged,omitempty"`
	Image       string              `yaml:"image,omitempty"`
	Ports       []string            `yaml:"ports,omitempty"`
	Environment map[string]string   `yaml:"environment,omitempty"`
	Volumes     []string            `yaml:"volumes,omitempty"`
	DependsOn   []string            `yaml:"depends_on,omitempty"`
	Networks    []string            `yaml:"networks,omitempty"`
	Devices     []map[string]string `yaml:"devices,omitempty"`
	CapAdd      []string            `yaml:"cap_add,omitempty"`
}

func ParseManifest(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var dc Manifest
	if err := yaml.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return &dc, nil
}

func ParseDockerCompose(filePath string) (*DockerCompose, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var dc DockerCompose
	if err := yaml.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return &dc, nil
}

func SaveDockerCompose(filePath string, dc any) error {
	data, err := yaml.Marshal(dc)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %v", err)
	}

	// Use os.WriteFile which will create or truncate the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

func CopyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write destination file: %v", err)
	}

	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
