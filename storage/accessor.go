package storage

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type Accessor interface {
	List() (PathList, error)
	Get(path string) (map[string]string, error)
	//Authenticate receives if existing token is still valid, according to the
	//TTL returned by the previous authentication attempt.  It should return the
	//new TTL. Authenticate must be called at some point before any calls to List
	//or Get.
	Authenticate(shouldRefresh bool) (TokenTTL, error)
}

const (
	typeUnknown int = iota
	typeVault
	typeOpsman
	typeCredhub
	typeTLS
)

type TokenTTL struct {
	//TTL is how much longer the token is valid for
	TTL time.Duration
	//Refreshable returns if the token should be refreshed. This is used
	// to calculate the shouldRefresh variable passed into Authenticate
	Refreshable bool
	//Last, if true, will cause the scheduler to not attempt any further
	//authentication calls.
	Last bool
}

const (
	TTLUnknown time.Duration = 0
	//TTLInfinite, if auth succeeds, means that no further renewal is necessary, as the auth will last forever
	TTLInfinite = time.Duration(0x7FFFFFFFFFFFFFFF)
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
