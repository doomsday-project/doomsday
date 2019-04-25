package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/doomsday-project/doomsday/server/auth"
	"github.com/doomsday-project/doomsday/server/notify"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Backends      []BackendConfig `yaml:"backends"`
	Server        APIConfig       `yaml:"server"`
	Notifications notify.Config   `yaml:"notifications"`
}

type APIConfig struct {
	Port    uint16 `yaml:"port"`
	LogFile string `yaml:"logfile"`
	TLS     struct {
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"tls"`
	Auth auth.Config `yaml:"auth"`
	Dev  struct {
		Mappings map[string]string `yaml:"mappings"`
	} `yaml:"dev"`
}

type BackendConfig struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
	//in minutes
	RefreshInterval int                    `yaml:"refresh_interval"`
	Properties      map[string]interface{} `yaml:"properties"`
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

	//Post Defaults
	for i, b := range conf.Backends {
		if b.RefreshInterval == 0 {
			conf.Backends[i].RefreshInterval = 30
		}
	}

	//Validation
	if conf.Server.Port < 0 || conf.Server.Port > 65535 {
		return nil, fmt.Errorf("Port number is invalid")
	}

	for _, b := range conf.Backends {
		if b.RefreshInterval <= 0 {
			return nil, fmt.Errorf("Refresh interval for backend must be greater than or equal to 0 - got %d", b.RefreshInterval)
		}
	}

	return &conf, nil
}
