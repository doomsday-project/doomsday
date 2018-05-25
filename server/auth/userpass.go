package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pborman/uuid"
	"github.com/thomasmmitchell/doomsday/duration"
)

type sessions struct {
	table   map[string]time.Time
	timeout time.Duration
	refresh bool
}

func (s *sessions) new() string {
	u := uuid.NewUUID()
	s.table[u.String()] = time.Now().Add(s.timeout)
	return u.String()
}

func (s *sessions) validate(sessionID string) bool {
	if expiry, found := s.table[sessionID]; found {
		if time.Now().Before(expiry) {
			if s.refresh {
				s.table[sessionID] = time.Now().Add(s.timeout)
			}
			return true
		}

		delete(s.table, sessionID)
	}

	return false
}

type Userpass struct {
	username string
	password string
	sessions sessions
}

func (u *Userpass) Configure(conf map[string]string) error {
	if conf == nil {
		return fmt.Errorf("No auth config provided")
	}

	if conf["username"] == "" {
		return fmt.Errorf("No username provided in userpass auth config")
	}

	if conf["password"] == "" {
		return fmt.Errorf("No password provided in userpass auth config")
	}

	timeout := 30 * time.Minute
	if conf["timeout"] != "" {
		var err error
		timeout, err = duration.Parse(conf["timeout"])
		if err != nil {
			return fmt.Errorf("Could not parse server timeout string")
		}
	}

	*u = Userpass{
		username: conf["username"],
		password: conf["password"],
		sessions: sessions{
			table:   map[string]time.Time{},
			timeout: timeout,
			refresh: conf["refresh"] != "" && conf["refresh"] != "false",
		},
	}

	return nil
}

func (u *Userpass) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		provided := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err = json.Unmarshal(body, &provided)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		if provided.Username != u.username || provided.Password != u.password {
			w.WriteHeader(401)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`+"\n", u.sessions.new())))
	}
}

func (u *Userpass) TokenHandler() TokenFunc {
	return func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Doomsday-Token")
			if u.sessions.validate(sessionID) {
				fn(w, r)
			} else {
				w.WriteHeader(401)
			}
		}
	}
}
