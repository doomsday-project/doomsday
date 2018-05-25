package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/thomasmmitchell/doomsday/storage"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Backend storage.Config `yaml:"backend"`
	Server  struct {
		Port    uint16 `yaml:"port"`
		LogFile string `yaml:"logfile"`
		TLS     struct {
			Cert string `yaml:"cert"`
			Key  string `yaml:"key"`
		} `yaml:"tls"`
		Auth struct {
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
	envPort, err := strconv.ParseUint(os.Getenv("PORT"), 10, 16)
	conf.Server.Port = uint16(envPort)
	err = yaml.Unmarshal(fileContents, &conf)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config (%s) as YAML: %s", path, err)
	}

	return &conf, nil
}
