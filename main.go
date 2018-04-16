package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
	"github.com/starkandwayne/goutils/ansi"
)

func main() {
	conf, err := parseConfig("ddayconfig.yml")
	if err != nil {
		bailWith(err.Error())
	}

	var backend Backend

	switch strings.ToLower(conf.Backend.Type) {
	case "vault":
		u, err := url.Parse(conf.Backend.Address)
		if err != nil {
			bailWith("Could not parse url (%s) in config: %s", u, err)
		}
		backend = &VaultBackend{
			Client: &vaultkv.Client{
				VaultURL:  u,
				AuthToken: conf.Backend.Auth["token"],
				Client: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				},
				//Trace: os.Stdout,
			},
		}

		if conf.Backend.BasePath == "" {
			conf.Backend.BasePath = "secret"
		}

	default:
		bailWith("Unrecognized backend type (%s)", conf.Backend.Type)
	}

	core := BackendCore{
		Backend:  backend,
		BasePath: conf.Backend.BasePath,
		Cache:    NewCache(),
	}
	err = core.Populate()
	if err != nil {
		bailWith("Failed to populate information: %s", err)
	}

	var outputList []Entry
	for _, key := range core.Cache.Keys() {
		value, found := core.Cache.Read(key)
		if !found {
			continue
		}

		outputList = append(outputList, Entry{
			Key:      key,
			NotAfter: value.NotAfter,
		})
	}

	sort.Slice(outputList, func(i, j int) bool { return outputList[i].NotAfter.Before(outputList[j].NotAfter) })

	for _, entry := range outputList {
		fmt.Printf("%s: %s\n", entry.Key, time.Until(entry.NotAfter).Truncate(time.Minute))
	}
}

type Entry struct {
	Key      string
	NotAfter time.Time
}

func bailWith(f string, a ...interface{}) {
	ansi.Fprintf(os.Stderr, fmt.Sprintf("@R{%s}\n", f), a...)
	os.Exit(1)
}
