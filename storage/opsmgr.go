package storage

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/storage/uaa"
)

type OmAccessor struct {
	client       *http.Client
	uaaClient    *uaa.Client
	url          *url.URL
	lock         sync.RWMutex
	clientID     string
	clientSecret string
	username     string
	password     string
	accessToken  string
	refreshToken string
	authType     uint64
}

type OmConfig struct {
	Address            string `yaml:"address"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	Auth               struct {
		GrantType    string `yaml:"grant_type"`
		Username     string `yaml:"username"`
		Password     string `yaml:"password"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"auth"`
}

type omAuthMetadata struct {
	renewalDeadline time.Time
}

func newOmClient(conf OmConfig) *http.Client {
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

func newOmAccessor(conf OmConfig) (*OmAccessor, omAuthMetadata, error) {
	metadata := omAuthMetadata{}
	u, err := url.Parse(conf.Address)
	if err != nil {
		return nil, metadata, fmt.Errorf("could not parse target url: %s", err)
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	var client = newOmClient(conf)
	var uaaClient = &uaa.Client{
		URL:               fmt.Sprintf("%s/uaa/oauth/token", u.String()),
		SkipTLSValidation: conf.InsecureSkipVerify,
	}

	var authType uint64

	switch conf.Auth.GrantType {
	case "client_credentials", "client credentials", "clientcredentials":
		authType = uaa.AuthClientCredentials
	case "resource_owner", "resource owner", "resourceowner", "password":
		authType = uaa.AuthPassword
	default:
		return nil, metadata, fmt.Errorf("Unknown auth grant_type `%s'", conf.Auth.GrantType)
	}

	ret := &OmAccessor{
		url:          u,
		uaaClient:    uaaClient,
		client:       client,
		clientID:     conf.Auth.ClientID,
		clientSecret: conf.Auth.ClientSecret,
		username:     conf.Auth.Username,
		password:     conf.Auth.Password,
		authType:     authType,
	}

	return ret, metadata, nil
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

//List attempts to list the paths in the ops manager that could have certs
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

func (v *OmAccessor) setTokens(accessToken, refreshToken string) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.accessToken = accessToken
	v.refreshToken = refreshToken
}

func (v *OmAccessor) getRefreshToken() string {
	v.lock.Lock()
	ret := v.refreshToken
	v.lock.Unlock()
	return ret
}

func (v *OmAccessor) Authenticate(last interface{}) (time.Duration, interface{}, error) {
	var authResp *uaa.AuthResponse
	var err error
	metadata := last.(omAuthMetadata)

	switch v.authType {
	case uaa.AuthClientCredentials:
		fmt.Fprintf(os.Stderr, "Performing client credentials auth for Ops Manager\n")
		authResp, err = v.uaaClient.ClientCredentials(v.clientID, v.clientSecret)

	case uaa.AuthPassword:
		attemptTime := time.Now()
		//The one second buffer is just so that we reduce the chance that we try
		// to renew the token just as the token is becoming unrenewable (and therefore err)
		if attemptTime.Add(1 * time.Second).Before(metadata.renewalDeadline) {
			fmt.Fprintf(os.Stderr, "Refreshing auth using refresh token for Ops Manager\n")
			authResp, err = v.uaaClient.Refresh(v.clientID, v.clientSecret, v.getRefreshToken())
		} else {
			fmt.Fprintf(os.Stderr, "Performing password auth for Ops Manager\n")
			authResp, err = v.uaaClient.Password(v.clientID, v.clientSecret, v.username, v.password)
		}

		if err == nil {
			metadata.renewalDeadline = attemptTime.Add(authResp.TTL)
		}

	default:
		panic("Unknown authType set in configServerAccessor")
	}
	if err != nil {
		return TTLUnknown, metadata, err
	}

	v.setTokens(authResp.AccessToken, authResp.RefreshToken)

	return authResp.TTL, metadata, nil
}

func (v *OmAccessor) opsmanAPI(path string) ([]byte, error) {
	u := *v.url
	u.Path = fmt.Sprintf("%s%s", u.Path, path)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return []byte{}, err
	}

	v.lock.RLock()
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.accessToken))
	v.lock.RUnlock()

	resp, err := v.client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("could not make api request to credentials endpoint: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		reqDump, _ := httputil.DumpRequest(req, true)
		respDump, _ := httputil.DumpResponse(resp, true)
		return []byte{}, fmt.Errorf("%s\n\n%s", reqDump, respDump)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return respBody, nil
}
