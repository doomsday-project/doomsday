package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/server"
	"github.com/thomasmmitchell/doomsday/storage"
)

func main() {
	conf, err := parseConfig("ddayconfig.yml")
	if err != nil {
		bailWith(err.Error())
	}

	var backend storage.Accessor

	switch strings.ToLower(conf.Backend.Type) {
	case "vault":
		u, err := url.Parse(conf.Backend.Address)
		if err != nil {
			bailWith("Could not parse url (%s) in config: %s", u, err)
		}
		backend = &storage.VaultAccessor{
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

	core := &doomsday.Core{
		Backend:  backend,
		BasePath: conf.Backend.BasePath,
		Cache:    doomsday.NewCache(),
	}
	server.Start(conf.Server, core)
}

func bailWith(f string, a ...interface{}) {
	ansi.Fprintf(os.Stderr, fmt.Sprintf("@R{%s}\n", f), a...)
	os.Exit(1)
}
