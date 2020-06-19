package storage

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/credhub-cli/credhub"
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

type configServerAuthMetadata struct {
	renewalDeadline time.Time
}

func newConfigServerAccessor(conf ConfigServerConfig) (
	*ConfigServerAccessor,
	configServerAuthMetadata,
	error,
) {
	metadata := configServerAuthMetadata{}
	c, err := credhub.New(
		conf.Address,
		credhub.SkipTLSValidation(conf.InsecureSkipVerify),
	)
	if err != nil {
		return nil, metadata, fmt.Errorf("Could not create config server client: %s", err)
	}
	c.Auth = &refreshTokenStrategy{APIClient: c.Client()}

	authURL, err := c.AuthURL()
	if err != nil {
		return nil, metadata, fmt.Errorf("Could not get auth endpoint: %s", err)
	}

	var authType uint64

	switch conf.Auth.GrantType {
	case "client_credentials", "client credentials", "clientcredentials":
		authType = uaa.AuthClientCredentials
	case "resource_owner", "resource owner", "resourceowner", "password":
		authType = uaa.AuthPassword
	case "none", "noop":
		authType = uaa.AuthNoop
	default:
		return nil, metadata, fmt.Errorf("Unknown auth grant_type `%s'", conf.Auth.GrantType)
	}

	ret := &ConfigServerAccessor{
		credhub: c,
		uaaClient: &uaa.Client{
			URL:               authURL,
			SkipTLSValidation: conf.InsecureSkipVerify,
		},
		authType:     authType,
		clientID:     conf.Auth.ClientID,
		clientSecret: conf.Auth.ClientSecret,
		username:     conf.Auth.Username,
		password:     conf.Auth.Password,
	}

	return ret, metadata, nil
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
		if certInterface, found := cred.Value.(map[string]interface{})["certificate"]; found && certInterface != nil {
			certAsString, isString := certInterface.(string)
			if isString {
				return map[string]string{"certificate": certAsString}, nil
			}
		}
	}

	return nil, nil
}

func (v *ConfigServerAccessor) Authenticate(last interface{}) (
	TTL time.Duration,
	next interface{},
	err error,
) {
	var authResp *uaa.AuthResponse
	metadata := last.(configServerAuthMetadata)

	switch v.authType {
	case uaa.AuthNoop:
		return TTLInfinite, metadata, nil

	case uaa.AuthClientCredentials:
		fmt.Fprintf(os.Stderr, "Performing client credentials auth for Credhub\n")
		authResp, err = v.uaaClient.ClientCredentials(v.clientID, v.clientSecret)

	case uaa.AuthPassword:
		attemptTime := time.Now()
		//The one second buffer is just so that we reduce the chance that we try
		// to renew the token just as the token is becoming unrenewable (and therefore err)
		if attemptTime.Add(1 * time.Second).Before(metadata.renewalDeadline) {
			fmt.Fprintf(os.Stderr, "Refreshing auth using refresh token for Credhub\n")
			authResp, err = v.uaaClient.Refresh(v.clientID, v.clientSecret, v.credhub.Auth.(*refreshTokenStrategy).RefreshToken())
		} else {
			fmt.Fprintf(os.Stderr, "Performing password auth for Credhub\n")
			authResp, err = v.uaaClient.Password(v.clientID, v.clientSecret, v.username, v.password)
		}

		if err == nil {
			metadata.renewalDeadline = attemptTime.Add(authResp.TTL)
		}

	default:
		panic("Unknown authType set in configServerAccessor")
	}
	if err != nil {
		return TTLUnknown, metadata, err
	}

	v.credhub.Auth.(*refreshTokenStrategy).SetTokens(authResp.AccessToken, authResp.RefreshToken)

	return authResp.TTL, metadata, nil
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
