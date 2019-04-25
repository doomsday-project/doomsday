package main

import (
	"github.com/doomsday-project/doomsday/server"
)

type serverCmd struct {
	ManifestPath *string
}

func (s *serverCmd) Run() error {
	conf, err := server.ParseConfig(*s.ManifestPath)
	if err != nil {
		return err
	}

	return server.Start(*conf)
}
