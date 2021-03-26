package storage

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
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
	CACerts            string `yaml:"ca_certs"`
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

func newOmClient(conf OmConfig) (*http.Client, error) {
	certPool, _ := x509.SystemCertPool()
	if conf.CACerts != "" {
		certPool = x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM([]byte(conf.CACerts))
		if !ok {
			return nil, fmt.Errorf("Could not parse provided CA certificates")
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.InsecureSkipVerify,
				RootCAs:            certPool,
			},
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
		},
	}, nil
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

	client, err := newOmClient(conf)
	if err != nil {
		return nil, metadata, err
	}

	certPool, _ := x509.SystemCertPool()
	if conf.CACerts != "" {
		certPool = x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM([]byte(conf.CACerts))
		if !ok {
			return nil, metadata, fmt.Errorf("Could not parse provided CA certificates")
		}
	}

	var uaaClient = &uaa.Client{
		URL:               fmt.Sprintf("%s/uaa/oauth/token", u.String()),
		SkipTLSValidation: conf.InsecureSkipVerify,
		CACerts:           certPool,
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
	pathDetails := strings.Split(path, "/")
	if len(pathDetails) != 2 {
		return map[string]string{}, fmt.Errorf("path '%s' does not properly specify a product guid and property reference", path)
	}

	var credentials struct {
		Cred struct {
			Type  string            `json:"type"`
			Value map[string]string `json:"value"`
		} `json:"credential"`
	}

	respBody, err := v.opsmanAPI(fmt.Sprintf("/api/v0/deployed/products/%s/credentials/%s", pathDetails[0], pathDetails[1]))
	if err != nil {
		if pathDetails[0] == "ops_manager" {
			return v.opsmanRootFallback(pathDetails[0], pathDetails[1])
		}
		if strings.HasPrefix(pathDetails[0], "p-bosh-") {
			return v.directorFallback(pathDetails[0], pathDetails[1])
		}
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
	path := "/api/v0/deployed/certificates"

	var certificateReference struct {
		Certificates []struct {
			Configurable      bool        `json:"configurable"`
			IsCa              bool        `json:"is_ca"`
			PropertyReference string      `json:"property_reference"`
			PropertyType      string      `json:"property_type"`
			ProductGUID       string      `json:"product_guid"`
			Location          string      `json:"location"`
			VariablePath      interface{} `json:"variable_path"`
			Issuer            string      `json:"issuer"`
			ValidFrom         time.Time   `json:"valid_from"`
			ValidUntil        time.Time   `json:"valid_until"`
		} `json:"certificates"`
	}

	respBody, err := v.opsmanAPI(path)
	if err != nil {
		return []string{}, err
	}

	err = json.Unmarshal(respBody, &certificateReference)
	if err != nil {
		return []string{}, fmt.Errorf("could not unmarshal certificates response: %s\nresponse: `%s`", err, respBody)
	}

	for _, cert := range certificateReference.Certificates {
		// Only get certs that are stored in opsman - leave credhub certs to use the
		// credhub backend
		if cert.Location == "ops_manager" {
			finalPaths = append(finalPaths, fmt.Sprintf("%s/%s", cert.ProductGUID, cert.PropertyReference))
		}
	}

	return finalPaths, nil
}

func (v *OmAccessor) directorFallback(guid string, property string) (map[string]string, error) {
	propertyRaw := strings.Split(property, ".")
	if len(propertyRaw) != 3 {
		return map[string]string{}, fmt.Errorf("property '%s' is not properly formatted", property)
	}
	property = propertyRaw[2]

	var settings struct {
		Products []map[string]interface{} `json:"products"`
	}

	respBody, err := v.opsmanAPI("/api/installation_settings")
	if err != nil {
		return map[string]string{}, err
	}

	err = json.Unmarshal(respBody, &settings)
	if err != nil {
		return map[string]string{}, fmt.Errorf("could not unmarshal certificates response: %s\nresponse: `%s`", err, respBody)
	}

	for _, product := range settings.Products {
		if product["guid"] == guid {
			propertyMap, isMap := product[property].(map[string]interface{})
			if !isMap {
				return map[string]string{}, fmt.Errorf("failed to unmarshal property %s from product in fallback api", property)
			}
			key, isMap := propertyMap["cert_pem"].(string)
			if !isMap {
				return map[string]string{}, fmt.Errorf("failed to unmarshal cert from property %s in fallback api", property)
			}
			return map[string]string{
				"cert_pem": key,
			}, nil
		}
	}

	return map[string]string{}, nil
}

func (v *OmAccessor) opsmanRootFallback(guid string, property string) (map[string]string, error) {
	propertyRaw := strings.Split(property, ".")
	if len(propertyRaw) != 4 {
		return map[string]string{}, fmt.Errorf("property '%s' is not properly formatted", property)
	}
	property = propertyRaw[2]
	certGUID := propertyRaw[3]

	var certificateAuthorities struct {
		CAs []struct {
			GUID        string    `json:"guid"`
			Issuer      string    `json:"issuer"`
			CreatedOn   time.Time `json:"created_on"`
			ExpiresOn   time.Time `json:"expires_on"`
			Active      bool      `json:"active"`
			CertPem     string    `json:"cert_pem"`
			NatsCertPem string    `json:"nats_cert_pem"`
		} `json:"certificate_authorities"`
	}

	respBody, err := v.opsmanAPI("/api/v0/certificate_authorities")
	if err != nil {
		return map[string]string{}, err
	}

	err = json.Unmarshal(respBody, &certificateAuthorities)
	if err != nil {
		return map[string]string{}, fmt.Errorf("could not unmarshal certificates response: %s\nresponse: `%s`", err, respBody)
	}

	for _, ca := range certificateAuthorities.CAs {
		if ca.GUID == certGUID {
			if property == "nats_client_ca" {
				return map[string]string{
					"cert_pem": ca.NatsCertPem,
				}, nil
			}
			if property == "root_ca" {
				return map[string]string{
					"cert_pem": ca.CertPem,
				}, nil
			}
		}
	}

	return map[string]string{}, nil
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
