package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func registerCommands(app *kingpin.Application) {
	serverCom := app.Command("server", "Start the doomsday server")
	cmdIndex["server"] = &serverCmd{
		configPath: serverCom.Flag("config", "The path to the config file").
			Short('c').
			Default("ddayconfig.yml").String(),
	}
}

func main() {
	var app = kingpin.New("doomsday", "Cert expiration tracker")
	registerCommands(app)

	commandName := kingpin.MustParse(app.Parse(os.Args[1:]))
	err := cmdIndex[commandName].Run()
	if err != nil {
		bailWith(err.Error())
	}
}

func bailWith(f string, a ...interface{}) {
	ansi.Fprintf(os.Stderr, fmt.Sprintf("@R{%s}\n", f), a...)
	os.Exit(1)
}
