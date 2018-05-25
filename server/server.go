package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/server/auth"
	"github.com/thomasmmitchell/doomsday/storage"
)

type server struct {
	Core *doomsday.Core
}

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
	fmt.Fprintf(logWriter, "Configuring targeted storage backend\n")

	var backend storage.Accessor
	switch strings.ToLower(conf.Backend.Type) {
	case "vault":
		backend, err = storage.NewVaultAccessor(&conf.Backend)
	case "opsmgr", "ops manager", "opsman", "opsmanager":
		backend, err = storage.NewOmAccessor(&conf.Backend)
	case "credhub", "configserver", "config server":
		backend, err = storage.NewConfigServer(&conf.Backend)
	default:
		err = fmt.Errorf("Unrecognized backend type (%s)", conf.Backend.Type)
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(logWriter, "Setting up doomsday core components\n")

	core := &doomsday.Core{
		Backend: backend,
	}

	core.SetCache(doomsday.NewCache())

	populate := func() {
		startedAt := time.Now()
		err := core.Populate()
		if err != nil {
			fmt.Fprintf(logWriter, "%s: Error populating cache: %s\n", time.Now(), err)
		}
		fmt.Printf("Populate took %s\n", time.Since(startedAt))
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

	fmt.Fprintf(logWriter, "Configuring frontend authentication\n")
	var authorizer auth.Authorizer
	switch conf.Server.Auth.Type {
	case "", "nop", "none":
		fmt.Fprintf(logWriter, "No server auth requested\n")
		authorizer = &auth.Nop{}
	case "userpass":
		fmt.Fprintf(logWriter, "userpass auth requested\n")
		authorizer = &auth.Userpass{}
	default:
		return fmt.Errorf("Unrecognized auth type `%s'", conf.Server.Auth.Type)
	}

	err = authorizer.Configure(conf.Server.Auth.Config)
	if err != nil {
		return fmt.Errorf("Error configuring auth: %s", err)
	}

	auth := authorizer.TokenHandler()
	router := mux.NewRouter()
	router.HandleFunc("/v1/auth", authorizer.LoginHandler()).Methods("POST")
	router.HandleFunc("/v1/cache", auth(getCache(core))).Methods("GET")
	router.HandleFunc("/v1/cache/refresh", auth(refreshCache(core))).Methods("POST")

	fmt.Fprintf(logWriter, "Beginning listening on port %d\n", conf.Server.Port)

	if conf.Server.TLS.Cert != "" || conf.Server.TLS.Key != "" {
		err = listenAndServeTLS(&conf, router)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", conf.Server.Port), router)
	}

	return err
}

func listenAndServeTLS(conf *Config, handler http.Handler) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Server.Port))
	if err != nil {
		return err
	}

	defer ln.Close()

	cert, err := tls.X509KeyPair([]byte(conf.Server.TLS.Cert), []byte(conf.Server.TLS.Key))
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(ln, &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{cert},
	})

	return http.Serve(tlsListener, handler)
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

		resp, err := json.Marshal(&doomsday.GetCacheResponse{Content: items})
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
