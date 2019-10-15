package storage

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
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
}

type VaultConfig struct {
	Address            string `yaml:"address"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	BasePath           string `yaml:"base_path"`
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
		//Trace: os.Stdout,
	}

	var shouldRenew bool
	var ttl time.Duration
	authType := vaultAuthToken
	if conf.Auth.RoleID != "" || conf.Auth.SecretID != "" {
		if conf.Auth.Token != "" {
			return nil, fmt.Errorf("Cannot provide both Token and AppRole authentication methods")
		}

		authType = vaultAuthApprole

		output, err := client.AuthApprole(conf.Auth.RoleID, conf.Auth.SecretID)
		if err != nil {
			return nil, fmt.Errorf("Error performing AppRole authentication: %s", err)
		}

		if output.Renewable {
			shouldRenew = true
			ttl = output.LeaseDuration
		}
	} else {
		//It's a token!
		info, err := client.TokenInfoSelf()
		if err != nil {
			return nil, fmt.Errorf("Error looking up token information: %s", err)
		}

		//Is it a renewable token?
		if info.Renewable {
			shouldRenew = true
			ttl = info.CreationTTL
			if ttl > 0 {
				//Give it an initial renew so that its remaining TTL is similar to its total TTL
				err = client.TokenRenewSelf()
				if err != nil {
					return nil, fmt.Errorf("Could not perform initial renewal of renewable token: %s", err)
				}
			}
		}
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
