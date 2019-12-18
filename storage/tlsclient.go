package storage

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net/url"
)

type TLSClientAccessor struct {
	hosts []string
}

type TLSClientConfig struct {
	Hosts []string `yaml:"hosts"`
}

func newTLSClientAccessor(conf TLSClientConfig) (*TLSClientAccessor, error) {
	ret := &TLSClientAccessor{}
	if len(conf.Hosts) == 0 {
		return nil, fmt.Errorf("No hosts list was specified in the configuration")
	}

	for _, host := range conf.Hosts {
		thisURL, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("The configured hosts list contained an invalid URL (%s): %s", host, err)
		}

		ret.hosts = append(ret.hosts, thisURL.String())
	}

	return ret, nil
}

func (t *TLSClientAccessor) List() (PathList, error) {
	ret := make(PathList, 0, len(t.hosts))
	for _, host := range t.hosts {
		ret = append(ret, host)
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

func (t *TLSClientAccessor) Authenticate(_ bool) (TokenTTL, error) {
	return TokenTTL{TTL: TTLInfinite}, nil
}
