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
	vaultAuthToken uint = iota
	vaultAuthApprole
)

type VaultAccessor struct {
	client   *vaultkv.KV
	basePath string
	authType uint
	//roleID and secretID are used for AppRole authentication
	roleID   string
	secretID string
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

type vaultAuthMetadata struct {
	renewalDeadline time.Time
}

func newVaultAccessor(conf VaultConfig) (*VaultAccessor, vaultAuthMetadata, error) {
	if !regexp.MustCompile("^.*://").MatchString(conf.Address) {
		conf.Address = fmt.Sprintf("https://%s", conf.Address)
	}

	metadata := vaultAuthMetadata{}

	u, err := url.Parse(conf.Address)
	if err != nil {
		return nil, metadata, fmt.Errorf("Could not parse url (%s) in config: %s", conf.Address, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, metadata, fmt.Errorf("Unsupported URL scheme `%s'", u.Scheme)
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

	authType := vaultAuthToken
	if conf.Auth.RoleID != "" || conf.Auth.SecretID != "" {
		if conf.Auth.Token != "" {
			return nil, metadata, fmt.Errorf("Cannot provide both Token and AppRole authentication methods")
		}

		authType = vaultAuthApprole
	} else {
		attemptTime := time.Now()
		tokenInfo, err := client.TokenInfoSelf()
		if err != nil {
			return nil, metadata, fmt.Errorf("Could not get token info: %s", err)
		}
		metadata.renewalDeadline = attemptTime.Add(tokenInfo.TTL)
	}

	return &VaultAccessor{
		client:   client.NewKV(),
		basePath: conf.BasePath,
		authType: authType,
		roleID:   conf.Auth.RoleID,
		secretID: conf.Auth.SecretID,
	}, metadata, nil
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

func (v *VaultAccessor) Authenticate(last interface{}) (
	time.Duration,
	interface{},
	error,
) {
	//TODO: Doooooo thiissssss
	var (
		ret TokenTTL
		err error
	)

	switch v.authType {
	case vaultAuthToken:
		ret, err = v.authToken(shouldRefresh)

	case vaultAuthApprole:
		if shouldRefresh {
			ret, err = v.authToken(true)
		} else {
			ret, err = v.authApprole(shouldRefresh)
		}
	}

	return ret, err
}

func (v *VaultAccessor) authToken(shouldRefresh bool) (TokenTTL, error) {
	info, err := v.client.Client.TokenInfoSelf()
	if err != nil {
		return TokenTTL{}, fmt.Errorf("Error looking up token information: %s", err)
	}

	var ttl time.Duration
	if info.ExpireTime.IsZero() {
		//Likely a root token
		ttl = TTLInfinite
	} else {
		ttl = time.Until(info.ExpireTime)
	}

	if ttl <= 0 {
		return TokenTTL{Last: true},
			fmt.Errorf("Token is expired")
	}

	//Is it a renewable token?
	if info.Renewable {
		//If so, renew it.
		err = v.client.Client.TokenRenewSelf()
		if err != nil {
			return TokenTTL{}, fmt.Errorf("Could not renew token: %s", err)
		}
		info, err := v.client.Client.TokenInfoSelf()
		if err != nil {
			return TokenTTL{}, fmt.Errorf("Error looking up token information after auth: %s", err)
		}
		return TokenTTL{TTL: info.TTL, Refreshable: info.Renewable}, nil
	}

	//If token is NOT renewable, say how much time is left
	return TokenTTL{TTL: ttl, Refreshable: false, Last: true}, nil
}

func (v *VaultAccessor) authApprole(shouldRenew bool) (TokenTTL, error) {
	output, err := v.client.Client.AuthApprole(v.roleID, v.secretID)
	if err != nil {
		return TokenTTL{}, fmt.Errorf("Error performing AppRole authentication: %s", err)
	}

	return TokenTTL{
		TTL:         output.LeaseDuration,
		Refreshable: output.Renewable,
	}, nil
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
