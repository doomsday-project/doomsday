package storage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type OmAccessor struct {
	Client *http.Client
	Host   string
	Scheme string
}

func newOmClient(conf *Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.InsecureSkipVerify,
			},
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
		},
	}
}

func NewOmAccessor(conf *Config) (*OmAccessor, error) {
	var client *http.Client
	var err error

	switch conf.Auth["grant_type"] {
	case "client_credentials", "client credentials", "clientcredentials":
		client, err = newClientCredentials(conf)
	case "resource_owner", "resource owner", "resourceowner", "password":
		client, err = newResourceOwner(conf)
	}

	if err != nil {
		return nil, err
	}

	u, err := url.Parse(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("could not parse target url: %s", err)
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	return &OmAccessor{
		Client: client,
		Host:   u.Host,
		Scheme: u.Scheme,
	}, nil
}

func newClientCredentials(conf *Config) (*http.Client, error) {
	config := &clientcredentials.Config{
		ClientID:     conf.Auth["client_id"],
		ClientSecret: conf.Auth["client_secret"],
		TokenURL:     fmt.Sprintf("%s/uaa/oauth/token", conf.Address),
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, newOmClient(conf))

	//This isn't necessary for the flow, but it let's us check if the ops manager
	// configuration is wrong ahead of time
	_, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error fetching token with client_credentials grant: %s", err)
	}

	return config.Client(ctx), nil
}

func newResourceOwner(conf *Config) (*http.Client, error) {
	config := &oauth2.Config{
		ClientID:     conf.Auth["client_id"],
		ClientSecret: conf.Auth["client_secret"],
		Endpoint: oauth2.Endpoint{
			TokenURL: fmt.Sprintf("%s/uaa/oauth/token", conf.Address),
			AuthURL:  fmt.Sprintf("%s/uaa/oauth/authorize", conf.Address),
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, newOmClient(conf))

	token, err := config.PasswordCredentialsToken(ctx, conf.Auth["username"], conf.Auth["password"])
	if err != nil {
		return nil, fmt.Errorf("Error fetching token with password grant: %s", err)
	}

	return config.Client(ctx, token), nil
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
func (v *OmAccessor) List() (PathList, error) {
	var finalPaths []string
	deployments, err := v.getDeployments()
	if err != nil {
		return []string{}, err
	}

	for _, deployment := range deployments {
		path := fmt.Sprintf("/api/v0/deployed/products/%s/credentials", deployment)

		var credentialReferences struct {
			Credentials []string `json:"credentials"`
		}

		respBody, err := v.opsmanAPI(path)
		if err != nil {
			return []string{}, err
		}

		err = json.Unmarshal(respBody, &credentialReferences)
		if err != nil {
			return []string{}, fmt.Errorf("could not unmarshal credentials response: %s\nresponse: `%s`", err, respBody)
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
		return []string{}, fmt.Errorf("could not unmarshal credentials response: %s\nresponse: `%s`", err, respBody)
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
		respDump, _ := httputil.DumpResponse(resp, true)
		return []byte{}, fmt.Errorf("%s", respDump)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return respBody, nil
}
