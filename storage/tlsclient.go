package storage

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net/url"
)

type TLSClientAccessor struct {
	Hosts []string
}

func NewTLSClientAccessor(conf *Config) (*TLSClientAccessor, error) {
	ret := &TLSClientAccessor{}
	hostsInterface, exists := conf.Config["hosts"]
	if !exists {
		return nil, fmt.Errorf("No hosts list was specified in the configuration")
	}

	hosts, isSlice := hostsInterface.([]interface{})
	if !isSlice {
		return nil, fmt.Errorf("The configured hosts key was not a list")
	}

	for _, hostInterface := range hosts {
		host, isString := hostInterface.(string)
		if !isString {
			return nil, fmt.Errorf("The configured hosts list contained a non-string (%v)", hostInterface)
		}
		thisURL, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("The configured hosts list contained an invalid URL (%s): %s", host, err)
		}

		ret.Hosts = append(ret.Hosts, thisURL.String())
	}

	return ret, nil
}

func (t *TLSClientAccessor) Get(path string) (map[string]string, error) {
	u, err := url.Parse(path)
	if err != nil {
		panic("TLS Client Accessor somehow couldn't parse a URL that it already checked")
	}

	if u.Host == "" {
		u, _ = url.Parse(fmt.Sprintf("garbage://%s", path))
	}

	if u.Port() == "" {
		u.Host = fmt.Sprintf("%s:443", u.Host)
	}

	conn, err := tls.Dial("tcp", u.Host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}

	certs := conn.ConnectionState().PeerCertificates
	ret := map[string]string{}
	if len(certs) != 0 {
		cert := certs[0].Raw
		pemCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		})

		ret["cert"] = string(pemCert)
	}

	return ret, nil
}

func (t *TLSClientAccessor) List() (PathList, error) {
	ret := make(PathList, 0, len(t.Hosts))
	for _, host := range t.Hosts {
		ret = append(ret, host)
	}

	return ret, nil
}
