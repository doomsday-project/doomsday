package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pborman/uuid"
)

var sessions = map[string]time.Time{}

func newSession() string {
	u := uuid.NewUUID()
	sessions[u.String()] = time.Now().Add(30 * time.Minute)
	return u.String()
}

func validateSession(sessionID string) bool {
	if expiry, found := sessions[sessionID]; found {
		if time.Now().Before(expiry) {
			return true
		}

		delete(sessions, sessionID)
	}

	return false
}

type userpassAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

func newUserpassAuth(conf map[string]string) (*userpassAuth, error) {
	if conf == nil {
		return nil, fmt.Errorf("No auth config provided")
	}

	if conf["username"] == "" {
		return nil, fmt.Errorf("No username provided in userpass auth config")
	}

	if conf["password"] == "" {
		return nil, fmt.Errorf("No password provided in userpass auth config")
	}

	return &userpassAuth{
		Username: conf["username"],
		Password: conf["password"],
	}, nil
}

func (u userpassAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	provided := userpassAuth{}
	err = json.Unmarshal(body, &provided)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	if provided.Username != u.Username || provided.Password != u.Password {
		w.WriteHeader(401)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`+"\n", newSession())))
}

type authorizer func(http.HandlerFunc) http.HandlerFunc

func nopAuth(fn http.HandlerFunc) http.HandlerFunc { return fn }

func sessionAuth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.Header.Get("X-Doomsday-Token")
		if validateSession(sessionID) {
			fn(w, r)
		} else {
			w.WriteHeader(401)
		}
	}
}
