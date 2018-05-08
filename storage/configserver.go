package storage

import (
	"fmt"

	"github.com/cloudfoundry-incubator/credhub-cli/credhub"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub/auth"
)

type ConfigServerAccessor struct {
	Credhub *credhub.CredHub
}

func NewConfigServer(conf *Config) (*ConfigServerAccessor, error) {
	var err error
	authStrat := auth.Noop
	switch conf.Auth["grant_type"] {
	case "client_credentials", "client credentials", "clientcredentials":
		authStrat = auth.UaaClientCredentials(
			conf.Auth["client_id"],
			conf.Auth["client_secret"],
		)

	case "resource_owner", "resource owner", "resourceowner", "password":
		authStrat = auth.UaaPassword(
			conf.Auth["client_id"],
			conf.Auth["client_secret"],
			conf.Auth["username"],
			conf.Auth["password"],
		)

	case "none", "noop": //The default is the noop builder
	default:
		err = fmt.Errorf("Unknown auth grant_type `%s'", conf.Auth["grant_type"])
	}

	if err != nil {
		return nil, err
	}

	c, err := credhub.New(
		conf.Address,
		credhub.SkipTLSValidation(conf.InsecureSkipVerify),
		credhub.Auth(authStrat),
	)
	if err != nil {
		return nil, fmt.Errorf("Could not create config server client: %s", err)
	}

	err = c.Auth.(*auth.OAuthStrategy).Login()
	if err != nil {
		return nil, fmt.Errorf("Could not authenticate with given credentials: %s", err)
	}

	return &ConfigServerAccessor{Credhub: c}, nil
}

//List attempts to get all of the paths in the config server
func (v *ConfigServerAccessor) List() (PathList, error) {
	paths, err := v.Credhub.FindByPath("/")
	if err != nil {
		return nil, fmt.Errorf("Could not get paths in config server: %s", err)
	}

	ret := make(PathList, 0, len(paths.Credentials))
	for _, entry := range paths.Credentials {
		ret = append(ret, entry.Name)
	}

	return ret, nil
}

func (v *ConfigServerAccessor) Get(path string) (map[string]string, error) {
	cred, err := v.Credhub.GetLatestVersion(path)
	if err != nil {
		return nil, err
	}

	if cred.Type == "certificate" {
		if certInterface, found := cred.Value.(map[string]interface{})["certificate"]; found {
			return map[string]string{"certificate": certInterface.(string)}, nil
		}
	}

	return nil, nil
}
