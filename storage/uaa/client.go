package uaa

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	URL               string
	SkipTLSValidation bool
}

func (c *Client) client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.SkipTLSValidation,
			},
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
		},
	}
}

func (c *Client) do(values url.Values) (*AuthResponse, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/oauth/token", c.URL),
		strings.NewReader(values.Encode()),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Could not authenticate: Status %d", resp.StatusCode)
	}

	type response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	r := response{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&r)
	if err != nil {
		return nil, err
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TTL:          time.Second * time.Duration(r.ExpiresIn),
	}, nil
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	TTL          time.Duration
}

func (c *Client) ClientCredentials(
	clientID,
	clientSecret string) (*AuthResponse, error) {

	return c.do(url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{clientID},
		"client_secret": []string{clientSecret},
	})
}

func (c *Client) Password(
	clientID,
	clientSecret,
	username,
	password string) (*AuthResponse, error) {

	return c.do(url.Values{
		"grant_type":    []string{"password"},
		"client_id":     []string{clientID},
		"client_secret": []string{clientSecret},
		"username":      []string{username},
		"password":      []string{password},
	})
}

func (c *Client) Refresh(
	clientID,
	clientSecret,
	refreshToken string) (*AuthResponse, error) {

	return c.do(url.Values{
		"grant_type":    []string{"refresh_token"},
		"client_id":     []string{clientID},
		"client_secret": []string{clientSecret},
		"refresh_token": []string{refreshToken},
	})
}
