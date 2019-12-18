package storage

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/credhub-cli/credhub"
	"github.com/doomsday-project/doomsday/storage/uaa"
)

type ConfigServerAccessor struct {
	credhub      *credhub.CredHub
	uaaClient    *uaa.Client
	authType     uint64
	clientID     string
	clientSecret string
	username     string
	password     string
}

type ConfigServerConfig struct {
	Address            string `yaml:"address"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	Auth               struct {
		GrantType    string `yaml:"grant_type"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		Username     string `yaml:"username"`
		Password     string `yaml:"password"`
	} `yaml:"auth"`
}

func newConfigServerAccessor(conf ConfigServerConfig) (*ConfigServerAccessor, error) {
	c, err := credhub.New(
		conf.Address,
		credhub.SkipTLSValidation(conf.InsecureSkipVerify),
	)
	if err != nil {
		return nil, fmt.Errorf("Could not create config server client: %s", err)
	}
	c.Auth = &refreshTokenStrategy{APIClient: c.Client()}

	authURL, err := c.AuthURL()
	if err != nil {
		return nil, fmt.Errorf("Could not get auth endpoint: %s", err)
	}

	var authType uint64

	switch conf.Auth.GrantType {
	case "client_credentials", "client credentials", "clientcredentials":
		authType = uaa.AuthClientCredentials
	case "resource_owner", "resource owner", "resourceowner", "password":
		authType = uaa.AuthPassword
	case "none", "noop":
		authType = uaa.AuthTypeNoop
	default:
		return nil, fmt.Errorf("Unknown auth grant_type `%s'", conf.Auth.GrantType)
	}

	ret := &ConfigServerAccessor{
		credhub: c,
		uaaClient: uaa.Client{
			URL:               authURL,
			SkipTLSValidation: conf.InsecureSkipVerify,
		},
		authType:     authType,
		clientID:     conf.Auth.ClientID,
		clientSecret: conf.Auth.ClientSecret,
		username:     conf.Auth.Username,
		password:     conf.Auth.Password,
	}

	return ret, nil
}

//List attempts to get all of the paths in the config server
func (v *ConfigServerAccessor) List() (PathList, error) {
	paths, err := v.credhub.FindByPath("/")
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
	cred, err := v.credhub.GetLatestVersion(path)
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

func (v *ConfigServerAccessor) Authenticate(shouldRefresh bool) (TokenTTL, error) {
	var authResp *uaa.AuthResponse
	var err error
	var refreshable bool

	switch v.authType {
	case uaa.AuthNoop:
		return &TokenTTL{
			TTL: TTLInfinite, 
			Last: true
		}, nil

	case uaa.AuthClientCredentials:
		fmt.Fprintf(os.Stderr, "Performing client credentials auth for Credhub\n")
		authResp, err = v.uaaClient.ClientCredentials(v.clientID, v.clientSecret)

	case uaa.AuthPassword:
		if shouldRefresh {
			fmt.Fprintf(os.Stderr, "Refreshing auth using refresh token for Credhub\n")
			authResp, err = v.uaaClient.Refresh(v.clientID, v.clientSecret, v.credhub.Auth.(refreshTokenStrategy).RefreshToken())
		} else {
			fmt.Fprintf(os.Stderr, "Performing password auth for Credhub\n")
			authResp, err = v.uaaClient.Password(v.clientID, v.clientSecret, v.username, v.password)
		}

		refreshable = true

	default:
		panic("Unknown authType set in configServerAccessor")
	}
	if err != nil {
		return TokenTTL{TTL: TTLUnknown}, err
	}

	v.credhub.Auth.(refreshTokenStrategy).SetTokens(authResp.AccessToken, authResp.RefreshToken)

	return &TokenTTL{TTL: authResp.TTL, Refreshable: refreshable}, nil
}

type refreshTokenStrategy struct {
	lock         sync.RWMutex
	accessToken  string
	refreshToken string
	APIClient    *http.Client
}

func (r *refreshTokenStrategy) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.AccessToken())
	return r.APIClient.Do(req)
}

func (r *refreshTokenStrategy) AccessToken() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.accessToken
}

func (r *refreshTokenStrategy) RefreshToken() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.refreshToken
}

func (r *refreshTokenStrategy) SetTokens(accessToken, refreshToken string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.accessToken = accessToken
	r.refreshToken = refreshToken
}
