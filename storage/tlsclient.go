package storage

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"
)

type TLSClientAccessor struct {
	hosts   []string
	timeout time.Duration
}

type TLSClientConfig struct {
	Hosts   []string `yaml:"hosts"`
	Timeout int      `yaml:"timeout"`
}

func newTLSClientAccessor(conf TLSClientConfig) (*TLSClientAccessor, interface{}, error) {
	ret := &TLSClientAccessor{}
	if len(conf.Hosts) == 0 {
		return nil, nil, fmt.Errorf("No hosts list was specified in the configuration")
	}

	for _, host := range conf.Hosts {
		thisURL, err := url.Parse(host)
		if err != nil {
			return nil, nil, fmt.Errorf("The configured hosts list contained an invalid URL (%s): %s", host, err)
		}

		ret.hosts = append(ret.hosts, thisURL.String())
	}

	if conf.Timeout < 0 {
		conf.Timeout = 0
	}

	if conf.Timeout != 0 {
		ret.timeout = time.Second * time.Duration(conf.Timeout)
	}

	return ret, nil, nil
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

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: t.timeout}, "tcp", u.Host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		//TODO: We should implement an actual warning system instead of just not
		// erroring
		//TODO: Also, we need to get the actual logger into these storage implementations
		fmt.Fprintf(os.Stderr, "Failed to connect to %s: %s\n", path, err)
		return nil, nil
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

func (t *TLSClientAccessor) Authenticate(_ interface{}) (time.Duration, interface{}, error) {
	return TTLInfinite, nil, nil
}
