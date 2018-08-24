package shout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

//Client has functions that handle interactions with SHOUT!
type Client struct {
	//Target is the URL that this client will hit with requests
	Target   string
	Username string
	Password string
	//HTTPClient is the net/http client that will be used to send requests.
	// If left nil, http.DefaultClient will be used instead
	HTTPClient *http.Client
	Trace      io.Writer
}

//EventIn is the input to PostEvent, and should contain information about the
// event to post to SHOUT!
type EventIn struct {
	//The topic name
	Topic string
	//A message about the event
	Message string
	//A URL relevent to the event
	Link string
	//The time that the event occurred
	OccurredAt time.Time
	//True if the event represents a "working" state. False if "broken"
	OK bool
	//Optional values to pass through to the user that can be used in the rules file
	Metadata map[string]string
}

func (c *Client) doRequest(method, path string, body []byte) (*http.Response, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(method,
		fmt.Sprintf("%s%s", c.Target, path),
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.Username, c.Password)

	if c.Trace != nil {
		b, _ := httputil.DumpRequestOut(req, true)
		c.Trace.Write(b)
		c.Trace.Write([]byte("\n"))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if c.Trace != nil {
		b, _ := httputil.DumpResponse(resp, true)
		c.Trace.Write(b)
		c.Trace.Write([]byte("\n"))
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SHOUT! returned non-2xx status code: %s", resp.Status)
	}

	return resp, nil
}

type eventRaw struct {
	OccurredAt int64  `json:"occurred-at"`
	ReportedAt int64  `json:"reported-at"`
	OK         bool   `json:"ok"`
	Message    string `json:"message"`
	Link       string `json:"link"`
}

type stateRaw struct {
	Name     string   `json:"name"`
	State    string   `json:"state"`
	Previous eventRaw `json:"previous"`
	First    eventRaw `json:"first"`
	Last     eventRaw `json:"last"`
}

//PostEvent sends the given event to SHOUT! to update the state of the topic.
// The event will send a message to notification backends configured by the
// rules of the SHOUT! backend if the state has changed
func (c *Client) PostEvent(e EventIn) error {
	jsonStruct := struct {
		Topic      string            `json:"topic"`
		Message    string            `json:"message"`
		Link       string            `json:"link"`
		OccurredAt int64             `json:"occurred-at"`
		OK         bool              `json:"ok"`
		Metadata   map[string]string `json:"metadata,omitempty"`
	}{
		Topic:      e.Topic,
		OK:         e.OK,
		Message:    e.Message,
		Link:       e.Link,
		OccurredAt: e.OccurredAt.Unix(),
		Metadata:   e.Metadata,
	}

	jBytes, _ := json.Marshal(&jsonStruct)

	_, err := c.doRequest("POST", "/events", jBytes)
	return err
}

//AnnouncementIn is the input to PostAnnouncement, containing information about
// the announcement event to send
type AnnouncementIn struct {
	//The name of the topic
	Topic string `json:"topic"`
	//The message to announce
	Message string `json:"message"`
	//A URL relevant to the announcement
	Link string `json:"link"`
}

//PostAnnouncement sends a message that goes to notification backends configured
// by the rules of the SHOUT! backend. This has no concept of a "working" or
// "broken" state, and so the message is always sent.
func (c *Client) PostAnnouncement(announcement AnnouncementIn) error {
	jBytes, _ := json.Marshal(&announcement)
	_, err := c.doRequest("POST", "/events", jBytes)
	return err
}
