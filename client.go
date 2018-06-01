package doomsday

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/thomasmmitchell/doomsday/server/auth"
)

type Client struct {
	Client *http.Client
	URL    url.URL
	Token  string
	Trace  io.Writer
}

func (c *Client) doRequest(
	method, path string,
	input, output interface{}) error {
	var client = c.Client
	if c.Client == nil {
		client = http.DefaultClient
	}
	var reqBody io.Reader
	if input != nil {
		reqBody = &bytes.Buffer{}
		jEncoder := json.NewEncoder(reqBody.(*bytes.Buffer))
		err := jEncoder.Encode(&input)
		if err != nil {
			panic("Could not encode object for request body")
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.URL.String(), path), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-Doomsday-Token", c.Token)

	if c.Trace != nil {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return err
		}
		_, err = c.Trace.Write(dump)
		if err != nil {
			return err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.Trace != nil {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		_, err = c.Trace.Write(dump)
		if err != nil {
			return err
		}
	}

	if (resp.StatusCode / 100) != 2 {
		return parseError(resp.StatusCode)
	}

	if output != nil {
		jDecoder := json.NewDecoder(resp.Body)
		err := jDecoder.Decode(&output)
		return err
	}

	return nil
}

//UserpassAuth attempts to authenticate with the doomsday server. If successful,
// the response is stored into the client
func (c *Client) UserpassAuth(username, password string) error {
	output := struct {
		Token string `json:"token"`
	}{}

	err := c.doRequest("POST", "/v1/auth", map[string]string{
		"username": username,
		"password": password,
	}, &output)

	c.Token = output.Token
	return err
}

type CacheItem struct {
	BackendName string `json:"backend_name"`
	Path        string `json:"path"`
	CommonName  string `json:"common_name"`
	NotAfter    int64  `json:"not_after"`
}

type CacheItems []CacheItem
type CacheItemFilter struct {
	Beyond *time.Duration
	Within *time.Duration
}

func (c CacheItems) Filter(filter CacheItemFilter) CacheItems {
	ret := make(CacheItems, 0, len(c))
	for _, v := range c {
		ret = append(ret, v)
	}

	if filter.Beyond != nil {
		cutoff := time.Now().Add(*filter.Beyond)
		for i, v := range ret {
			if cutoff.Before(time.Unix(v.NotAfter, 0)) {
				ret = ret[i:]
				break
			}
		}
	}

	if filter.Within != nil {
		cutoff := time.Now().Add(*filter.Within)
		for i, v := range ret {
			if cutoff.Before(time.Unix(v.NotAfter, 0)) {
				ret = ret[:i]
				break
			}
		}
	}

	return ret
}

type GetCacheResponse struct {
	Content CacheItems `json:"content"`
}

//GetCache gets the cache list
func (c *Client) GetCache() (CacheItems, error) {
	resp := GetCacheResponse{}
	err := c.doRequest("GET", "/v1/cache", nil, &resp)
	return resp.Content, err
}

//RefreshCache makes a request to asynchronously refresh the server cache
func (c *Client) RefreshCache() error {
	return c.doRequest("POST", "/v1/cache/refresh", nil, nil)
}

type InfoResponse struct {
	Version  string        `json:"version"`
	AuthType auth.AuthType `json:"auth_type"`
}

func (c *Client) Info() (*InfoResponse, error) {
	resp := InfoResponse{}
	err := c.doRequest("GET", "/v1/info", nil, &resp)
	return &resp, err
}
