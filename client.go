package doomsday

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	Client *http.Client
	URL    url.URL
	Token  string
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

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	//dump, err := httputil.DumpResponse(resp, true)
	//if err != nil {
	//	return err
	//}
	//fmt.Println(string(dump))
	if (resp.StatusCode / 100) != 2 {
		return fmt.Errorf("Returned non-200 response code")
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

//GetCache gets the cache list
func (c *Client) GetCache() ([]CacheItem, error) {
	ret := []CacheItem{}
	err := c.doRequest("GET", "/v1/cache", nil, &ret)
	return ret, err
}
