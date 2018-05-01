package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
	"github.com/gorilla/mux"
	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/storage"
	"golang.org/x/oauth2"
)

type server struct {
	Core *doomsday.Core
}

//TODO: Refactor this into helper functions
func Start(conf Config) error {
	var err error
	logWriter := os.Stderr
	if conf.Server.LogFile != "" {
		logWriter, err = os.OpenFile(conf.Server.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Could not open log file for writing: %s", err)
		}
	}

	fmt.Fprintf(logWriter, "Initializing server\n")

	var backend storage.Accessor
	switch strings.ToLower(conf.Backend.Type) {
	case "vault":
		u, err := url.Parse(conf.Backend.Address)
		if err != nil {
			return fmt.Errorf("Could not parse url (%s) in config: %s", u, err)
		}

		backend = &storage.VaultAccessor{
			Client: &vaultkv.Client{
				VaultURL:  u,
				AuthToken: conf.Backend.Auth["token"],
				Client: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: conf.Backend.InsecureSkipVerify,
						},
					},
				},
				//Trace: os.Stdout,
			},
		}

		if conf.Backend.BasePath == "" {
			conf.Backend.BasePath = "secret"
		}
	case "opsmgr":
		u, err := url.Parse(conf.Backend.Address)
		if err != nil {
			return fmt.Errorf("Could not parse url (%s) in config: %s", u, err)
		}

		config := &oauth2.Config{
			ClientID:     "opsman",
			ClientSecret: "",
			Endpoint: oauth2.Endpoint{
				TokenURL: "https://10.213.9.1/uaa/oauth/token",
			},
		}

		httpclient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: conf.Backend.InsecureSkipVerify,
				},
				Dial: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
			},
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpclient)

		token, err := config.PasswordCredentialsToken(ctx, conf.Backend.Auth["username"], conf.Backend.Auth["password"])
		if err != nil {
			return fmt.Errorf("token could not be retrieved from target url: %s", err)
		}

		url, err := url.Parse(conf.Backend.Address)
		if err != nil {
			return fmt.Errorf("could not parse target url: %s", err)
		}

		if url.Scheme == "" {
			url.Scheme = "https"
		}

		backend = &storage.OmAccessor{
			Client: config.Client(ctx, token),
			Host:   url.Host,
			Scheme: url.Scheme,
		}
	default:
		return fmt.Errorf("Unrecognized backend type (%s)", conf.Backend.Type)
	}

	core := &doomsday.Core{
		Backend:  backend,
		BasePath: conf.Backend.BasePath,
	}

	core.SetCache(doomsday.NewCache())

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
	switch conf.Server.Auth.Type {
	case "":
		shouldNotAuth = true
	case "userpass":
		authHandler, err = newUserpassAuth(conf.Server.Auth.Config)
	default:
		err = fmt.Errorf("Unrecognized auth type `%s'", conf.Server.Auth.Type)
	}
	if err != nil {
		return fmt.Errorf("Error setting up auth: %s", err)
	}

	fmt.Fprintf(logWriter, "Auth configured with type `%s'\n", conf.Server.Auth.Type)

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

	fmt.Fprintf(logWriter, "Beginning listening on port %d\n", conf.Server.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", conf.Server.Port), router)
}

func getCache(core *doomsday.Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		data := core.Cache().Map()
		items := make([]doomsday.CacheItem, 0, len(data))
		for k, v := range data {
			items = append(items, doomsday.CacheItem{
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
