package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/thomasmmitchell/doomsday/server/auth"
	"github.com/thomasmmitchell/doomsday/storage"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Backend storage.Config `yaml:"backend"`
	Server  APIConfig      `yaml:"server"`
}

type APIConfig struct {
	Port    uint16 `yaml:"port"`
	LogFile string `yaml:"logfile"`
	TLS     struct {
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"tls"`
	Auth auth.Config `yaml:"auth"`
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

	//Set defaults
	conf := Config{
		Server: APIConfig{
			Port: 8111,
		},
	}

	if os.Getenv("PORT") != "" {
		envPort, err := strconv.ParseUint(os.Getenv("PORT"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Could not convert port to int")
		}

		conf.Server.Port = uint16(envPort)
	}

	//Read config
	err = yaml.Unmarshal(fileContents, &conf)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config (%s) as YAML: %s", path, err)
	}

	//Validation
	if conf.Server.Port < 0 || conf.Server.Port > 65535 {
		return nil, fmt.Errorf("Port number is invalid")
	}

	return &conf, nil
}
