package storage

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
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

	schemeRegex := regexp.MustCompile("^.+?://.+")
	for _, host := range conf.Hosts {

		toParse := host
		if !schemeRegex.Match([]byte(host)) {
			toParse = "garbage://" + host
		}

		thisURL, err := url.Parse(toParse)
		if err != nil {
			return nil, nil, fmt.Errorf("The configured hosts list contained an invalid URL (%s): %s", host, err)
		}

		if thisURL.Port() == "" {
			thisURL.Host = thisURL.Host + ":443"
		}

		ret.hosts = append(ret.hosts, thisURL.Host)
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

func (t *TLSClientAccessor) Get(host string) (map[string]string, error) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: t.timeout}, "tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		//TODO: We should implement an actual warning system instead of just not
		// erroring
		//TODO: Also, we need to get the actual logger into these storage implementations
		fmt.Fprintf(os.Stderr, "Failed to connect to %s: %s\n", host, err)
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
