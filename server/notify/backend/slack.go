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
	Webhook string `yaml:"webhook"`
}

type Slack struct {
	webhook string
}

func newSlackBackend(c SlackConfig) (*Slack, error) {
	if c.Webhook == "" {
		return nil, fmt.Errorf("No webhook provided")
	}

	if _, err := url.Parse(c.Webhook); err != nil {
		return nil, fmt.Errorf("Webhook not parsable as URL")
	}
	return &Slack{webhook: c.Webhook}, nil
}

func (s Slack) Send(m Message) error {
	var toSend string
	for _, v := range m {
		switch part := v.(type) {
		case MText:
			toSend += s.quoteMeta(part.Text)
		case MLink:
			if part.Text == "" {
				part.Text = part.Link
			}
			toSend += fmt.Sprintf("<%s|%s>", part.Link, s.quoteMeta(part.Text))
		}
	}

	body, err := json.Marshal(&map[string]string{"text": toSend})
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

func (Slack) quoteMeta(s string) string {
	s = strings.Replace(s, "&", "&amp;", -1)
	s = strings.Replace(s, "<", "&lt;", -1)
	s = strings.Replace(s, ">", "&gt;", -1)
	s = strings.Replace(s, "\r\n", "\\n", -1)
	s = strings.Replace(s, "\n", "\\n", -1)
	return s
}
