package server

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/thomasmmitchell/doomsday/storage"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Backend storage.Config `yaml:"backend"`
	Server  struct {
		Port    int    `yaml:"port"`
		LogFile string `yaml:"logfile"`
		Auth    struct {
			Type   string            `yaml:"type"`
			Config map[string]string `yaml:"config"`
		} `yaml:"auth"`
	} `yaml:"server"`
}

func ParseConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Could not open config at `%s': %s", path, err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read from config (%s): %s", path, err)
	}

	conf := Config{}
	err = yaml.Unmarshal(fileContents, &conf)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config (%s) as YAML: %s", path, err)
	}

	return &conf, nil
}
