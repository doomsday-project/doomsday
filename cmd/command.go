package main

type command interface {
	Run() error
}

var cmdIndex = map[string]command{}
