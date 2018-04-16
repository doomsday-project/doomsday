package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

type config struct {
	Backend backendConfig `yaml:"backend"`
}

type backendConfig struct {
	Type     string            `yaml:"type"`
	Address  string            `yaml:"address"`
	Auth     map[string]string `yaml:"auth"`
	BasePath string            `yaml:"base_path"`
}

func parseConfig(path string) (*config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Could not open config at `%s': %s", path, err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read from config (%s): %s", path, err)
	}

	conf := config{}
	err = yaml.Unmarshal(fileContents, &conf)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config (%s) as YAML: %s", path, err)
	}

	return &conf, nil
}
