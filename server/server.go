package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/thomasmmitchell/doomsday"
)

type Config struct {
	Port    int    `yaml:"port"`
	LogFile string `yaml:"logfile"`
	Auth    struct {
		Type   string            `yaml:"type"`
		Config map[string]string `yaml:"config"`
	} `yaml:"auth"`
}

type server struct {
	Core *doomsday.Core
}

func Start(conf Config, core *doomsday.Core) error {
	var err error
	logWriter := os.Stderr
	if conf.LogFile != "" {
		logWriter, err = os.OpenFile(conf.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Could not open log file for writing: %s", err)
		}
	}

	fmt.Fprintf(logWriter, "Initializing server\n")

	populate := func() {
		err := core.Populate()
		if err != nil {
			fmt.Fprintf(logWriter, "%s: Error populating cache: %s\n", time.Now(), err)
		}
	}

	go func() {
		populate()
		interval := time.NewTicker(time.Hour)
		defer interval.Stop()
		for range interval.C {
			populate()
		}
	}()

	fmt.Fprintf(logWriter, "Began asynchronous cache population\n")

	var shouldNotAuth bool
	var authHandler http.Handler
	switch conf.Auth.Type {
	case "":
		shouldNotAuth = true
	case "userpass":
		authHandler, err = newUserpassAuth(conf.Auth.Config)
	default:
		err = fmt.Errorf("Unrecognized auth type `%s'", conf.Auth.Type)
	}
	if err != nil {
		return fmt.Errorf("Error setting up auth: %s", err)
	}

	fmt.Fprintf(logWriter, "Auth configured with type `%s'\n", conf.Auth.Type)

	var auth authorizer
	router := mux.NewRouter()
	if shouldNotAuth {
		auth = nopAuth
	} else {
		router.Handle("/v1/auth", authHandler).Methods("POST")
		auth = sessionAuth
	}
	router.HandleFunc("/v1/cache", auth(getCache(core))).Methods("GET")
	router.HandleFunc("/v1/cache/refresh", auth(refreshCache(core))).Methods("POST")

	fmt.Fprintf(logWriter, "Beginning listening on port %d\n", conf.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), router)
}

func getCache(core *doomsday.Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type item struct {
			Path       string `json:"path"`
			CommonName string `json:"common_name"`
			NotAfter   int64  `json:"not_after"`
		}

		data := core.Cache.Map()
		items := make([]item, 0, len(data))
		for k, v := range data {
			items = append(items, item{
				Path:       k,
				CommonName: v.Subject.CommonName,
				NotAfter:   v.NotAfter.Unix(),
			})
		}

		sort.Slice(items, func(i, j int) bool { return items[i].NotAfter < items[j].NotAfter })

		resp, err := json.Marshal(&items)
		if err != nil {
			w.WriteHeader(500)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(resp)
		}
	}
}

func refreshCache(core *doomsday.Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go core.Populate()
		w.WriteHeader(204)
	}
}
