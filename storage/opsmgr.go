package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

type OmAccessor struct {
	Client *http.Client
	Host   string
	Scheme string
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

	respBody, err := v.opsmanAPI(path)
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
	var finalPaths []string
	deployments, err := v.getDeployments()
	if err != nil {
		return []string{}, err
	}

	for _, deployment := range deployments {
		path = fmt.Sprintf("/api/v0/deployed/products/%s/credentials", deployment)

		var credentialReferences struct {
			Credentials []string `json:"credentials"`
		}

		respBody, err := v.opsmanAPI(path)
		if err != nil {
			return []string{}, err
		}

		err = json.Unmarshal(respBody, &credentialReferences)
		if err != nil {
			return []string{}, fmt.Errorf("could not unmarshal credentials response: %s", err)
		}

		for _, cred := range credentialReferences.Credentials {
			finalPaths = append(finalPaths, fmt.Sprintf("/api/v0/deployed/products/%s/credentials/%s", deployment, cred))
		}
	}

	return finalPaths, nil
}

func (v *OmAccessor) getDeployments() ([]string, error) {
	path := fmt.Sprintf("/api/v0/deployed/products")
	respBody, err := v.opsmanAPI(path)
	if err != nil {
		return []string{}, err
	}
	var rawDeployments []struct {
		InstallationName string `json:"installation_name"`
		GUID             string `json:"guid"`
		Type             string `json:"type"`
		ProductVersion   string `json:"product_version"`
	}

	err = json.Unmarshal(respBody, &rawDeployments)
	if err != nil {
		return []string{}, fmt.Errorf("could not unmarshal credentials response: %s", err)
	}

	var deployments []string
	for _, deployment := range rawDeployments {
		deployments = append(deployments, deployment.GUID)
	}

	return deployments, nil
}

func (v *OmAccessor) opsmanAPI(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return []byte{}, err
	}

	req.URL.Scheme = v.Scheme
	req.URL.Host = v.Host

	resp, err := v.Client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("could not make api request to credentials endpoint: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, err := httputil.DumpResponse(resp, true)
		return []byte{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return respBody, nil
}
