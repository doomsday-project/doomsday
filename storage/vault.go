package storage

import "github.com/cloudfoundry-community/vaultkv"

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
func (v *VaultAccessor) List(path string) ([]string, error) {
	return v.Client.List(path)
}
