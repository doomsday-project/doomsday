package main

import (
	"fmt"
	"os"
)

type command interface {
	Run() error
}

var cmdIndex = map[string]command{}

func init() {

}

//GLOBALS
var (
	configPath = app.Flag("config", "Path to the config file").
		Short('c').
		Default(fmt.Sprintf("%s/.dday", os.Getenv("HOME"))).String()
)
