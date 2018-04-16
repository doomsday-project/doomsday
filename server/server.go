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

	router := mux.NewRouter()
	router.HandleFunc("/v1/cache", getCache(core)).Methods("GET")
	router.HandleFunc("/v1/cache/refresh", refreshCache(core)).Methods("POST")

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
			w.WriteHeader(200)
			w.Write(resp)
		}
	}
}

func refreshCache(core *doomsday.Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go core.Populate()
		w.WriteHeader(200)
	}
}
