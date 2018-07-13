package auth

import (
	"fmt"
	"net/http"

	yaml "gopkg.in/yaml.v2"
)

type Authorizer interface {
	LoginHandler() http.HandlerFunc
	TokenHandler() TokenFunc
	Identifier() AuthType
}
type Config struct {
	Type       string                 `yaml:"type"`
	Properties map[string]interface{} `yaml:"properties"`
}
type TokenFunc func(http.HandlerFunc) http.HandlerFunc

type AuthType string

const (
	typeUnknown int = iota
	typeNone
	typeUserpass
)

func NewAuth(conf Config) (Authorizer, error) {
	properties, err := yaml.Marshal(&conf.Properties)
	t := resolveType(conf.Type)

	if t == typeUnknown {
		return nil, fmt.Errorf("Unrecognized auth type `%s'", conf.Type)
	}

	var c interface{}

	switch t {
	case typeNone:
		return NewNop(NopConfig{})
	case typeUserpass:
		c = &UserpassConfig{}
		err = yaml.Unmarshal(properties, c.(*UserpassConfig))
	}

	if err != nil {
		return nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var a Authorizer
	switch t {
	case typeUserpass:
		a, err = NewUserpass(*c.(*UserpassConfig))
	}

	return a, err
}

func resolveType(t string) int {
	switch t {
	case "", "nop", "none":
		return typeNone
	case "userpass":
		return typeUserpass
	default:
		return typeUnknown
	}
}
