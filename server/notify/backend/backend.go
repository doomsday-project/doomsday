package backend

import (
	"fmt"
	"strings"

	"github.com/thomasmmitchell/doomsday/server/logger"
	yaml "gopkg.in/yaml.v2"
)

type Backend interface {
	OK() error
	Soon() error
	Expired() error
}

type Config struct {
	Type       string                 `yaml:"type"`
	Properties map[string]interface{} `yaml:"properties"`
}

type BackendUniversalConfig struct {
	DoomsdayURL string
	Logger      *logger.Logger
}

const (
	typeUnknown int = iota
	typeSlack
	typeShout
)

func New(conf Config, uni BackendUniversalConfig) (Backend, error) {
	properties, err := yaml.Marshal(&conf.Properties)
	if err != nil {
		panic("Could not re-marshal into YAML")
	}

	t := resolveType(strings.ToLower(conf.Type))
	if t == typeUnknown {
		return nil, fmt.Errorf("Unrecognized backend type (%s)", conf.Type)
	}

	var c interface{}

	switch t {
	case typeSlack:
		c = &SlackConfig{}
		err = yaml.Unmarshal(properties, c.(*SlackConfig))
	case typeShout:
		c = &ShoutConfig{}
		err = yaml.Unmarshal(properties, c.(*ShoutConfig))
	}

	if err != nil {
		return nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var backend Backend
	switch t {
	case typeSlack:
		backend, err = newSlackBackend(*c.(*SlackConfig), uni)
	case typeShout:
		backend, err = newShoutBackend(*c.(*ShoutConfig), uni)
	}

	return backend, err
}

func resolveType(t string) int {
	switch t {
	case "slack":
		return typeSlack
	case "shout", "shout!":
		return typeShout
	default:
		return typeUnknown
	}
}

const (
	msgOK      = "No certs are expiring soon"
	msgSoon    = "Warning! There are certs expiring soon"
	msgExpired = "AHHH! There are expired certs!"
)
