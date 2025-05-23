package utils

import (
	"bytes"
	"fmt"
	"os/exec"
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
