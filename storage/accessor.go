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
	//Authenticate receives metadata returned from the last run of a call to
	//authenticate. It is guaranteed to receive the value that was returned by
	//its constructor on the first run. It should return the new TTL, any
	//metadata to send to the next run, and an error if one occurred.
	//Authenticate must be called at some point before any calls to List or Get.
	Authenticate(last interface{}) (TTL time.Duration, nextMetadata interface{}, err error)
}

const (
	typeUnknown int = iota
	typeVault
	typeOpsman
	typeCredhub
	typeTLS
)

const (
	TTLUnknown time.Duration = 0
	//TTLInfinite means that no further renewal is necessary, as the auth will
	//last forever
	TTLInfinite = time.Duration(0x7FFFFFFFFFFFFFFF)
)

//NewAccessor generates an accessor of the provided type, configured with the
//provided configuration object. returns the Accessor, the struct to be passed
//to the accessor's first auth call, and an error if one occurred.
func NewAccessor(accessorType string, conf map[string]interface{}) (
	Accessor,
	interface{},
	error,
) {
	properties, err := yaml.Marshal(&conf)
	if err != nil {
		panic("Could not re-marshal into YAML")
	}

	t := resolveType(strings.ToLower(accessorType))
	if t == typeUnknown {
		return nil, nil, fmt.Errorf("Unrecognized backend type (%s)", accessorType)
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
		return nil, nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var backend Accessor
	var firstAuth interface{}
	switch t {
	case typeVault:
		backend, firstAuth, err = newVaultAccessor(*c.(*VaultConfig))
	case typeOpsman:
		backend, firstAuth, err = newOmAccessor(*c.(*OmConfig))
	case typeCredhub:
		backend, firstAuth, err = newConfigServerAccessor(*c.(*ConfigServerConfig))
	case typeTLS:
		backend, firstAuth, err = newTLSClientAccessor(*c.(*TLSClientConfig))
	}

	return backend, firstAuth, err
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
