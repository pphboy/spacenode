package ymlutils

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func ParseYAML[T any](path string) (T, error) {
	var data T
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return data, err
	}
	err = yaml.Unmarshal(file, &data)
	return data, err
}
