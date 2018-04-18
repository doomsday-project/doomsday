package storage

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
)

type VaultAccessor struct {
	Client   *vaultkv.Client
	BasePath string
}

//Get attempts to get the secret stored at the requested backend path and
// return it as a map.
func (v *VaultAccessor) Get(path string) (map[string]string, error) {
	ret := make(map[string]string)
	err := v.Client.Get(path, &ret)
	return ret, err
}

//List attempts to list the paths directly under the given path
func (v *VaultAccessor) List(path string) (PathList, error) {
	var leaves []string
	list, err := v.Client.List(path)
	if err != nil {
		return nil, err
	}

	for _, val := range list {
		if !strings.HasSuffix(val, "/") {
			leaves = append(leaves, canonizePath(fmt.Sprintf("%s/%s", path, val)))
		} else {
			rList, err := v.List(canonizePath(fmt.Sprintf("%s/%s", path, val)))
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
