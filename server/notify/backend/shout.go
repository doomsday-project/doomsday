package backend

import (
	"fmt"
	"net/url"
	"time"

	shout "github.com/thomasmmitchell/go-shout"
)

type ShoutConfig struct {
	URL      string `yaml:"url"`
	Topic    string `yaml:"topic"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Shout struct {
	client         shout.Client
	topic          string
	doomsdayDomain string
}

func newShoutBackend(c ShoutConfig, uni BackendUniversalConfig) (*Shout, error) {
	if c.URL == "" {
		return nil, fmt.Errorf("No URL provided")
	}

	if c.Topic == "" {
		return nil, fmt.Errorf("No topic name provided")
	}

	if c.Username == "" {
		return nil, fmt.Errorf("No username provided")
	}

	if c.Password == "" {
		return nil, fmt.Errorf("No password provided")
	}

	if u, err := url.Parse(c.URL); err != nil || u.Host == "" {
		return nil, fmt.Errorf("URL not parsable")
	}
	return &Shout{
		client: shout.Client{
			Target:   c.URL,
			Username: c.Username,
			Password: c.Password,
			Trace:    uni.Logger,
		},
		topic:          c.Topic,
		doomsdayDomain: uni.DoomsdayURL,
	}, nil
}

func (s Shout) OK() error {
	return s.client.PostEvent(shout.EventIn{
		Topic:      s.topic,
		Message:    msgOK,
		Link:       s.doomsdayDomain,
		OccurredAt: time.Now(),
		OK:         true,
	})
}

func (s Shout) Soon() error {
	return s.client.PostEvent(shout.EventIn{
		Topic:      s.topic,
		Message:    msgSoon,
		Link:       s.doomsdayDomain,
		OccurredAt: time.Now(),
		OK:         false,
	})
}

func (s Shout) Expired() error {
	return s.client.PostEvent(shout.EventIn{
		Topic:      s.topic,
		Message:    msgExpired,
		Link:       s.doomsdayDomain,
		OccurredAt: time.Now(),
		OK:         false,
	})
}
