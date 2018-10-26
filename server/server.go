package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/server/auth"
	"github.com/thomasmmitchell/doomsday/server/logger"
	"github.com/thomasmmitchell/doomsday/server/manager"
	"github.com/thomasmmitchell/doomsday/server/notify"
	"github.com/thomasmmitchell/doomsday/storage"
)

func Start(conf Config) error {
	var err error

	log := logger.NewLogger(os.Stderr)
	if conf.Server.LogFile != "" {
		var logTarget *os.File
		logTarget, err = os.OpenFile(conf.Server.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Could not open log file for writing: %s", err)
		}

		log = logger.NewLogger(logTarget)
	}

	log.WriteF("Initializing server")
	log.WriteF("Configuring targeted storage backends")

	sources := make([]manager.Source, 0, len(conf.Backends))
	for _, b := range conf.Backends {
		backendName := b.Name
		if backendName == "" {
			backendName = b.Type
		}

		log.WriteF("Configuring backend `%s' of type `%s'", b.Name, b.Type)
		thisBackend, err := storage.NewAccessor(b.Type, b.Properties)
		if err != nil {
			return fmt.Errorf("Error configuring backend `%s': %s", b.Name, err)
		}

		thisCore := doomsday.Core{Backend: thisBackend}
		thisCore.SetCache(doomsday.NewCache())

		sources = append(sources,
			manager.Source{
				Core:     &thisCore,
				Name:     backendName,
				Interval: time.Duration(b.RefreshInterval) * time.Minute,
			},
		)
	}

	manager := manager.NewSourceManager(sources, log)
	manager.BackgroundScheduler()

	log.WriteF("Began asynchronous cache population")
	log.WriteF("Configuring frontend authentication")

	authorizer, err := auth.NewAuth(conf.Server.Auth)
	if err != nil {
		return err
	}

	if conf.Notifications.Schedule.Type != "" {
		err = notify.NotifyFrom(conf.Notifications, manager, log)
		if err != nil {
			return fmt.Errorf("Error setting up notifications: %s", err)
		}

		log.WriteF("Notifications configured")
	}

	auth := authorizer.TokenHandler()
	router := mux.NewRouter()
	router.HandleFunc("/v1/info", getInfo(authorizer.Identifier())).Methods("GET")
	router.HandleFunc("/v1/auth", authorizer.LoginHandler()).Methods("POST")
	router.HandleFunc("/v1/cache", auth(getCache(manager))).Methods("GET")
	router.HandleFunc("/v1/cache/refresh", auth(refreshCache(manager))).Methods("POST")

	log.WriteF("Beginning listening on port %d", conf.Server.Port)

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

func getInfo(authType auth.AuthType) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(struct {
			Version  string `json:"version"`
			AuthType string `json:"auth_type"`
		}{
			Version:  doomsday.Version,
			AuthType: string(authType),
		})
		if err != nil {
			panic("Could not marshal info into json")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func getCache(manager *manager.SourceManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		items := manager.Data()
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

func refreshCache(manager *manager.SourceManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go manager.RefreshAll()
		w.WriteHeader(204)
	}
}
