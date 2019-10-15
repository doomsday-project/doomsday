package storage

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/credhub-cli/credhub"
	"github.com/doomsday-project/doomsday/storage/uaa"
)

type ConfigServerAccessor struct {
	credhub *credhub.CredHub
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
	var err error
	var authResp *uaa.AuthResponse

	c, err := credhub.New(
		conf.Address,
		credhub.SkipTLSValidation(conf.InsecureSkipVerify),
	)
	if err != nil {
		return nil, fmt.Errorf("Could not create config server client: %s", err)
	}

	authURL, err := c.AuthURL()
	if err != nil {
		return nil, fmt.Errorf("Could not get auth endpoint: %s", err)
	}

	//fmt.Printf("AuthURL: %s\n", authURL)

	uaaClient := uaa.Client{
		URL:               authURL,
		SkipTLSValidation: conf.InsecureSkipVerify,
	}

	var isClientCredentials bool

	switch conf.Auth.GrantType {
	case "client_credentials", "client credentials", "clientcredentials":
		fmt.Println("Performing client credentials grant auth")
		isClientCredentials = true
		authResp, err = uaaClient.ClientCredentials(
			conf.Auth.ClientID,
			conf.Auth.ClientSecret,
		)

	case "resource_owner", "resource owner", "resourceowner", "password":
		fmt.Println("Performing password grant auth")
		authResp, err = uaaClient.Password(
			conf.Auth.ClientID,
			conf.Auth.ClientSecret,
			conf.Auth.Username,
			conf.Auth.Password,
		)

	case "none", "noop": //The default is the noop builder
	default:
		err = fmt.Errorf("Unknown auth grant_type `%s'", conf.Auth.GrantType)
	}
	if err != nil {
		return nil, err
	}

	fmt.Println("Auth complete")

	c, err = credhub.New(
		conf.Address,
		credhub.SkipTLSValidation(conf.InsecureSkipVerify),
	)

	c.Auth = &refreshTokenStrategy{
		ClientID:              conf.Auth.ClientID,
		ClientSecret:          conf.Auth.ClientSecret,
		Username:              conf.Auth.Username,
		Password:              conf.Auth.Password,
		UAAClient:             &uaaClient,
		APIClient:             c.Client(),
		IsClientCredentials:   isClientCredentials,
		lastSuccessfulRefresh: time.Now(),
		TTL:                   authResp.TTL,
	}

	c.Auth.(*refreshTokenStrategy).SetTokens(authResp.AccessToken, authResp.RefreshToken)

	refreshInterval := authResp.TTL / 2
	fmt.Fprintf(os.Stderr, "Refreshing Credhub token every %s\n", refreshInterval)
	go func() {
		for range time.Tick(refreshInterval) {
			err = c.Auth.(*refreshTokenStrategy).Refresh()
			if err != nil {
				fmt.Printf("Could not refresh token: %s", err)
			}
		}
	}()

	return &ConfigServerAccessor{credhub: c}, nil
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

type refreshTokenStrategy struct {
	lock                  sync.RWMutex
	accessToken           string
	refreshToken          string
	lastSuccessfulRefresh time.Time
	ClientID              string
	ClientSecret          string
	Username              string
	Password              string
	IsClientCredentials   bool
	UAAClient             *uaa.Client
	APIClient             *http.Client
	TTL                   time.Duration
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

func (r *refreshTokenStrategy) Refresh() error {
	var authResp *uaa.AuthResponse
	var err error
	attemptTime := time.Now()
	if r.IsClientCredentials {
		fmt.Fprintf(os.Stderr, "Refreshing client credentials auth for Credhub\n")
		authResp, err = r.UAAClient.ClientCredentials(r.ClientID, r.ClientSecret)
	} else {
		if time.Since(r.lastSuccessfulRefresh) > r.TTL {
			fmt.Fprintf(os.Stderr, "Refreshing password auth for Credhub\n")
			authResp, err = r.UAAClient.Password(r.ClientID, r.ClientSecret, r.Username, r.Password)
		} else {
			fmt.Fprintf(os.Stderr, "Refreshing auth using refresh token for Credhub\n")
			authResp, err = r.UAAClient.Refresh(r.ClientID, r.ClientSecret, r.RefreshToken())
		}
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Credhub token refresh was successful\n")
	r.lastSuccessfulRefresh = attemptTime
	r.TTL = authResp.TTL

	r.SetTokens(authResp.AccessToken, authResp.RefreshToken)

	return nil
}

func (r *refreshTokenStrategy) SetTokens(accessToken, refreshToken string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.accessToken = accessToken
	r.refreshToken = refreshToken
}
