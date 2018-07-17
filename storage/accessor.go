package storage

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Accessor interface {
	List() (PathList, error)
	Get(path string) (map[string]string, error)
}

const (
	typeUnknown int = iota
	typeVault
	typeOpsman
	typeCredhub
	typeTLS
)

func NewAccessor(accessorType string, conf map[string]interface{}) (Accessor, error) {
	properties, err := yaml.Marshal(&conf)
	if err != nil {
		panic("Could not re-marshal into YAML")
	}

	t := resolveType(strings.ToLower(accessorType))
	if t == typeUnknown {
		return nil, fmt.Errorf("Unrecognized backend type (%s)", accessorType)
	}

	var c interface{}

	switch t {
	case typeVault:
		c = &VaultConfig{}
		err = yaml.Unmarshal(properties, c.(*VaultConfig))
	case typeOpsman:
		c = &OmConfig{}
		err = yaml.Unmarshal(properties, c.(*OmConfig))
	case typeCredhub:
		c = &ConfigServerConfig{}
		err = yaml.Unmarshal(properties, c.(*ConfigServerConfig))
	case typeTLS:
		c = &TLSClientConfig{}
		err = yaml.Unmarshal(properties, c.(*TLSClientConfig))
	}

	if err != nil {
		return nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var backend Accessor
	switch t {
	case typeVault:
		backend, err = newVaultAccessor(*c.(*VaultConfig))
	case typeOpsman:
		backend, err = newOmAccessor(*c.(*OmConfig))
	case typeCredhub:
		backend, err = newConfigServerAccessor(*c.(*ConfigServerConfig))
	case typeTLS:
		backend, err = newTLSClientAccessor(*c.(*TLSClientConfig))
	}

	return backend, err
}

func resolveType(t string) int {
	switch t {
	case "vault":
		return typeVault
	case "opsmgr", "ops manager", "opsman", "opsmanager":
		return typeOpsman
	case "credhub", "configserver", "config server":
		return typeCredhub
	case "tls", "tlsclient":
		return typeTLS
	default:
		return typeUnknown
	}
}
