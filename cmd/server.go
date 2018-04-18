package main

import (
	"github.com/thomasmmitchell/doomsday/server"
)

type serverCmd struct {
	configPath *string
}

func (s *serverCmd) Run() error {
	conf, err := server.ParseConfig(*s.configPath)
	if err != nil {
		return err
	}

	return server.Start(*conf)
}
