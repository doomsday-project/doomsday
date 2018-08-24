package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SlackConfig struct {
	Webhook  string `yaml:"webhook"`
	NotifyOK bool   `yaml:"notify_ok"`
}

type Slack struct {
	webhook  string
	topic    string
	notifyOK bool
}

func newSlackBackend(c SlackConfig, uni BackendUniversalConfig) (*Slack, error) {
	if c.Webhook == "" {
		return nil, fmt.Errorf("No webhook provided")
	}

	if _, err := url.Parse(c.Webhook); err != nil {
		return nil, fmt.Errorf("Webhook not parsable as URL")
	}
	return &Slack{
		webhook: c.Webhook,
		topic:   fmt.Sprintf("%s<%s>%s", slackQuoteMeta("doomsday: ("), uni.Logger, "): "),
	}, nil
}

func (s Slack) OK() error {
	var err error
	if s.notifyOK {
		err = s.send(msgOK)
	}
	return err
}

func (s Slack) Soon() error {
	return s.send(msgSoon)
}

func (s Slack) Expired() error {
	return s.send(msgExpired)
}

func (s Slack) send(msg string) error {
	body, err := json.Marshal(&map[string]string{
		"text": s.topic + slackQuoteMeta(msg),
	})
	if err != nil {
		panic("We tried to send a nil message")
	}
	r, err := http.NewRequest("POST", s.webhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Error when making request: %s", err)
	}

	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("Error sending request: %s", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Status non-2xx: %d", resp.StatusCode)
	}

	return nil
}

func slackQuoteMeta(s string) string {
	s = strings.Replace(s, "&", "&amp;", -1)
	s = strings.Replace(s, "<", "&lt;", -1)
	s = strings.Replace(s, ">", "&gt;", -1)
	s = strings.Replace(s, "\r\n", "\\n", -1)
	s = strings.Replace(s, "\n", "\\n", -1)
	return s
}
