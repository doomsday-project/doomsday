package storage

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
)

const (
	vaultAuthToken = iota
	vaultAuthApprole
)

type VaultAccessor struct {
	client   *vaultkv.KV
	basePath string
	authType string
}

type VaultConfig struct {
	Address            string `yaml:"address"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	BasePath           string `yaml:"base_path"`
	Trace              bool   `yaml:"trace"`
	Auth               struct {
		Token    string `yaml:"token"`
		RoleID   string `yaml:"role_id"`
		SecretID string `yaml:"secret_id"`
	} `yaml:"auth"`
}

func newVaultAccessor(conf VaultConfig) (*VaultAccessor, error) {
	if !regexp.MustCompile("^.*://").MatchString(conf.Address) {
		conf.Address = fmt.Sprintf("https://%s", conf.Address)
	}

	u, err := url.Parse(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("Could not parse url (%s) in config: %s", conf.Address, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("Unsupported URL scheme `%s'", u.Scheme)
	}

	if conf.BasePath == "" {
		conf.BasePath = "secret/"
	}

	var tracer io.Writer
	if conf.Trace {
		//I'm already tracer
		tracer = os.Stdout
	}

	client := &vaultkv.Client{
		VaultURL:  u,
		AuthToken: conf.Auth.Token,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: conf.InsecureSkipVerify,
				},
				MaxIdleConnsPerHost: runtime.NumCPU(),
			},
		},
		Trace: tracer,
	}

	var shouldRenew bool
	var ttl time.Duration
	authType := vaultAuthToken
	if conf.Auth.RoleID != "" || conf.Auth.SecretID != "" {
		if conf.Auth.Token != "" {
			return nil, fmt.Errorf("Cannot provide both Token and AppRole authentication methods")
		}

		authType = vaultAuthApprole
	} else {
	}

	if shouldRenew && ttl > 0 {
		renewalInterval := ttl / 2
		fmt.Printf("Renewing Vault token every %s\n", renewalInterval)
		go func() {
			lastSuccessfulRefresh := time.Now()

			for range time.Tick(renewalInterval) {
				attemptTime := time.Now()
				var err error
				if time.Since(lastSuccessfulRefresh) > ttl {
					if authType == vaultAuthApprole {
						fmt.Println("Renewing Vault token using AppRole authentication")
						_, err = client.AuthApprole(conf.Auth.RoleID, conf.Auth.SecretID)
					} else {
						fmt.Printf("Token is expired - no way to get new token for Vault. Stopping further renewal attempts.")
						return
					}
				} else {
					fmt.Println("Renewing Vault token using self-renewal")
					err = client.TokenRenewSelf()
				}

				if err != nil {
					fmt.Printf("Failed to renew token to Vault: %s\n", err)
				} else {
					lastSuccessfulRefresh = attemptTime
					fmt.Println("Successfully renewed Vault token")
				}
			}
		}()
	} else {
		fmt.Printf("Will not renew Vault token because ")
		if !shouldRenew {
			fmt.Printf("token is not renewable\n")
		} else if ttl <= 0 {
			fmt.Printf("token never expires\n")
		}
	}

	return &VaultAccessor{
		client:   client.NewKV(),
		basePath: conf.BasePath,
	}, nil

}

//Get attempts to get the secret stored at the requested backend path and
// return it as a map.
func (v *VaultAccessor) Get(path string) (map[string]string, error) {
	ret := make(map[string]string)
	_, err := v.client.Get(path, &ret, nil)
	if err != nil {
		//This might be worth checking to see if
		// 1. The mount is v2
		// 2. The secret metadata exists
		// 3. The latest version is deleted
		// But for now, if we listed it, this is probably why we'd get a 404.
		if vaultkv.IsNotFound(err) {
			err = nil
		}
	}
	return ret, err
}

//List attempts to list all the paths under the configured base path
func (v *VaultAccessor) List() (PathList, error) {
	return v.list(v.basePath)
}

func (v *VaultAccessor) list(path string) (PathList, error) {
	var leaves []string
	list, err := v.client.List(path)
	if err != nil {
		return nil, err
	}

	for _, val := range list {
		if !strings.HasSuffix(val, "/") {
			leaves = append(leaves, canonizePath(fmt.Sprintf("%s/%s", path, val)))
		} else {
			rList, err := v.list(canonizePath(fmt.Sprintf("%s/%s", path, val)))
			if err != nil {
				return nil, err
			}
			leaves = append(leaves, rList...)
		}
	}

	return leaves, nil
}

func (v *VaultAccessor) Authenticate(shouldRefresh bool) (TokenTTL, error) {
	switch v.authType {
	case vaultAuthToken:
		//It's a token!
		info, err := v.client.TokenInfoSelf()
		if err != nil {
			return nil, fmt.Errorf("Error looking up token information: %s", err)
		}

		var ttl time.Duration
		if info.ExpireTime.IsZero() {
			//Likely a root token
			ttl = ttl.Infinite
		} else {
			ttl = time.Until(info.ExpireTime)
		}

		if ttl <= 0 {
			return TokenTTL{Last: true},
				fmt.Errorf("Token is expired.")
		}

		//Is it a renewable token?
		if info.Renewable {
			//If so, renew it.
			err = client.TokenRenewSelf()
			if err != nil {
				return TokenTTL{}, fmt.Errorf("Could not renew token: %s", err)
			}
			info, err := client.TokenInfoSelf()
			return TokenTTL{TTL: info.TTL, Refreshable: info.Renewable}, nil
		}

		//If token is NOT renewable, say how much time is left
		return TokenTTL{TTL: ttl, Refreshable: false, Last: true}, nil

	case vaultAuthApprole:
		if shouldRefresh {

		}
		output, err := client.AuthApprole(conf.Auth.RoleID, conf.Auth.SecretID)
		if err != nil {
			return TokenTTL{}, fmt.Errorf("Error performing AppRole authentication: %s", err)
		}

		if output.Renewable {
			shouldRenew = true
			ttl = output.LeaseDuration
		}
	}
}

func canonizePath(path string) string {
	pathChunks := strings.Split(path, "/")
	for i := 0; i < len(pathChunks); i++ {
		if pathChunks[i] == "" {
			pathChunks = append(pathChunks[:i], pathChunks[i+1:]...)
			i--
		}
	}
	return strings.Join(pathChunks, "/")
}
