package backend

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Backend interface {
	Send(Message) error
}

const (
	typeUnknown int = iota
	typeSlack
)

func New(backendType string, conf map[string]interface{}) (Backend, error) {
	properties, err := yaml.Marshal(&conf)
	if err != nil {
		panic("Could not re-marshal into YAML")
	}

	t := resolveType(strings.ToLower(backendType))
	if t == typeUnknown {
		return nil, fmt.Errorf("Unrecognized backend type (%s)", backendType)
	}

	var c interface{}

	switch t {
	case typeSlack:
		c = &SlackConfig{}
		err = yaml.Unmarshal(properties, c.(*SlackConfig))
	}

	if err != nil {
		return nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var backend Backend
	switch t {
	case typeSlack:
		backend, err = newSlackBackend(*c.(*SlackConfig))
	}

	return backend, err
}

func resolveType(t string) int {
	switch t {
	case "slack":
		return typeSlack
	default:
		return typeUnknown
	}
}
