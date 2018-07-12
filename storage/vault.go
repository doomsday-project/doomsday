package storage

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
)

type VaultAccessor struct {
	Client   *vaultkv.Client
	BasePath string
}

func NewVaultAccessor(conf *Config) (*VaultAccessor, error) {
	u, err := url.Parse(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("Could not parse url (%s) in config: %s", u, err)
	}

	basePath := "secret/"
	if confBasePath, found := conf.Config["base_path"]; found {
		basePath = fmt.Sprintf("%v/", confBasePath)
	}

	return &VaultAccessor{
		Client: &vaultkv.Client{
			VaultURL:  u,
			AuthToken: conf.Auth["token"],
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: conf.InsecureSkipVerify,
					},
				},
			},
			//Trace: os.Stdout,
		},
		BasePath: basePath,
	}, nil

}

//Get attempts to get the secret stored at the requested backend path and
// return it as a map.
func (v *VaultAccessor) Get(path string) (map[string]string, error) {
	ret := make(map[string]string)
	err := v.Client.Get(path, &ret)
	return ret, err
}

//List attempts to list all the paths under the configured base path
func (v *VaultAccessor) List() (PathList, error) {
	return v.list(v.BasePath)
}

func (v *VaultAccessor) list(path string) (PathList, error) {
	var leaves []string
	list, err := v.Client.List(path)
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
