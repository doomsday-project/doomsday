package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

type OmAccessor struct {
	Client   *http.Client
	BasePath string
}

//Get attempts to get the secret stored at the requested backend path and
// return it as a map.
func (v *OmAccessor) Get(path string) (map[string]string, error) {
	var credentials struct {
		Cred struct {
			Type  string            `json:"type"`
			Value map[string]string `json:"value"`
		} `json:"credential"`
	}

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return credentials.Cred.Value, err
	}

	req.URL.Scheme = "https" //fixme
	req.URL.Host = "10.213.9.1"

	resp, err := v.Client.Do(req)
	if err != nil {
		return map[string]string{}, fmt.Errorf("could not make api request to credentials endpoint: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, err := httputil.DumpResponse(resp, true)
		return map[string]string{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]string{}, err
	}

	err = json.Unmarshal(respBody, &credentials)
	if err != nil {
		return map[string]string{}, fmt.Errorf("could not unmarshal credentials response: %s", err)
	}
	return credentials.Cred.Value, nil
}

//List attempts to list the paths directly under the given path
func (v *OmAccessor) List(path string) (PathList, error) {
	deployedGUID := "cf-32403a409e48e697b084" //fixme!
	path = fmt.Sprintf("/api/v0/deployed/products/%s/credentials", deployedGUID)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return []string{}, err
	}

	req.URL.Scheme = "https" //fixme
	req.URL.Host = "10.213.9.1"

	resp, err := v.Client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("could not make api request to credentials endpoint: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, err := httputil.DumpResponse(resp, true)
		return []string{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}

	var credentialReferences struct {
		Credentials []string `json:"credentials"`
	}

	err = json.Unmarshal(respBody, &credentialReferences)
	if err != nil {
		return []string{}, fmt.Errorf("could not unmarshal credentials response: %s", err)
	}

	var finalPaths []string
	for _, cred := range credentialReferences.Credentials {
		finalPaths = append(finalPaths, fmt.Sprintf("/api/v0/deployed/products/%s/credentials/%s", deployedGUID, cred))
	}

	return finalPaths, nil
}
