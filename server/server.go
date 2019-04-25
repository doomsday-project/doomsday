package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/doomsday-project/doomsday/server/auth"
	"github.com/doomsday-project/doomsday/server/logger"
	"github.com/doomsday-project/doomsday/storage"
	"github.com/doomsday-project/doomsday/version"
	"github.com/gorilla/mux"
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

	sources := make([]Source, 0, len(conf.Backends))
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

		thisCore := Core{Backend: thisBackend, Name: backendName}
		thisCore.SetCache(NewCache())

		sources = append(sources,
			Source{
				Core:     &thisCore,
				Interval: time.Duration(b.RefreshInterval) * time.Minute,
			},
		)
	}

	manager := NewSourceManager(sources, log)
	manager.BackgroundScheduler()

	log.WriteF("Began asynchronous cache population")
	log.WriteF("Configuring frontend authentication")

	authorizer, err := auth.NewAuth(conf.Server.Auth)
	if err != nil {
		return err
	}

	if conf.Notifications.Schedule.Type != "" {
		err = NotifyFrom(conf.Notifications, manager, log)
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

	if len(conf.Server.Dev.Mappings) > 0 {
		for file, servePath := range conf.Server.Dev.Mappings {
			servePath = "/" + strings.TrimPrefix(servePath, "/")
			log.WriteF("Serving %s at %s", file, servePath)
			router.HandleFunc(servePath, serveDevFile(file)).Methods("GET")
		}
	} else {
		for path, value := range webStatics {
			router.HandleFunc(path, serveFile(value.Content, value.MIMEType)).Methods("GET")
		}
	}

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
			Version:  version.Version,
			AuthType: string(authType),
		})
		if err != nil {
			panic("Could not marshal info into json")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func getCache(manager *SourceManager) func(w http.ResponseWriter, r *http.Request) {
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

func refreshCache(manager *SourceManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go manager.RefreshAll()
		w.WriteHeader(204)
	}
}

func serveFile(content, mimeType string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mimeType)
		w.WriteHeader(200)
		w.Write([]byte(content))
	}
}

func serveDevFile(filepath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(filepath)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("Could not serve file: %s", filepath)))
		}

		contents, err := ioutil.ReadAll(f)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Could not read contents of file"))
		}

		contentType := "text/plain"
		if strings.HasSuffix(filepath, ".html") {
			contentType = "text/html"
		} else if strings.HasSuffix(filepath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(filepath, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(filepath, ".svg") {
			contentType = "image/svg+xml"
		}

		w.Header().Set("Content-Type", contentType)

		w.WriteHeader(200)
		w.Write(contents)
		f.Close()
	}
}
