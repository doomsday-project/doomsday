package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pborman/uuid"
)

//AuthUserpass is the identifier returned by Nop.Userpass
const AuthUserpass = "Userpass"

type sessions struct {
	table   map[string]time.Time
	lock    sync.RWMutex
	timeout time.Duration
	refresh bool
}

func (s *sessions) new() string {
	u := uuid.NewUUID()
	s.lock.Lock()
	s.table[u.String()] = time.Now().Add(s.timeout)
	s.lock.Unlock()
	return u.String()
}

func (s *sessions) validate(sessionID string) bool {
	s.lock.RLock()
	expiry, found := s.table[sessionID]
	s.lock.RUnlock()
	if found {
		if time.Now().Before(expiry) {
			if s.refresh {
				s.lock.Lock()
				s.table[sessionID] = time.Now().Add(s.timeout)
				s.lock.Unlock()
			}
			return true
		}

		s.lock.Lock()
		delete(s.table, sessionID)
		s.lock.Unlock()
	}

	return false
}

type Userpass struct {
	username string
	password string
	sessions sessions
}

type UserpassConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  int    `yaml:"timeout"`
	Refresh  bool   `yaml:"refresh"`
}

func NewUserpass(conf UserpassConfig) (*Userpass, error) {
	if conf.Username == "" {
		return nil, fmt.Errorf("No username provided in userpass auth config")
	}

	if conf.Password == "" {
		return nil, fmt.Errorf("No password provided in userpass auth config")
	}

	if conf.Timeout == 0 {
		conf.Timeout = 30
	}

	return &Userpass{
		username: conf.Username,
		password: conf.Password,
		sessions: sessions{
			table:   map[string]time.Time{},
			timeout: time.Duration(conf.Timeout) * time.Minute,
			refresh: conf.Refresh,
		},
	}, nil
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

func (_ *Userpass) Identifier() AuthType {
	return AuthUserpass
}
