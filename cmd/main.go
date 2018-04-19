package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/thomasmmitchell/doomsday"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func registerCommands(app *kingpin.Application) {
	serverCom := app.Command("server", "Start the doomsday server")
	cmdIndex["server"] = &serverCmd{
		ManifestPath: serverCom.Flag("manifest", "The path to the server manifest").
			Short('m').
			Default("ddayconfig.yml").String(),
	}

	targetCom := app.Command("target", "Set a doomsday server to target")
	cmdIndex["target"] = &targetCmd{
		Name:    targetCom.Arg("name", "The name of the target").String(),
		Address: targetCom.Arg("address", "The address to set for this target").String(),
		SkipVerify: targetCom.Flag("insecure-skip-verify", "Skip TLS cert validation for this backend").
			Short('k').Bool(),
	}

	_ = app.Command("targets", "Print out configured targets")
	cmdIndex["targets"] = &targetsCmd{}

	loginCom := app.Command("login", "Auth to the doomsday server").Alias("auth")
	cmdIndex["login"] = &loginCmd{
		Username: loginCom.Flag("username", "The username to log in as").
			Short('u').String(),
		Password: loginCom.Flag("password", "The password to log in with").
			Short('p').String(),
	}
	cmdIndex["auth"] = cmdIndex["login"]

}

var app = kingpin.New("doomsday", "Cert expiration tracker")
var cliConf *CLIConfig
var target *CLITarget
var client *doomsday.Client

func main() {
	registerCommands(app)

	app.HelpFlag.Short('h')

	commandName := kingpin.MustParse(app.Parse(os.Args[1:]))
	cmd, found := cmdIndex[commandName]
	if !found {
		panic(fmt.Sprintf("Unregistered command %s", commandName))
	}

	if _, isServerCommand := cmd.(*serverCmd); !isServerCommand {
		var err error
		cliConf, err = loadConfig(*configPath)
		if err != nil {
			bailWith("Could not load CLI config from `%s': %s", *configPath, err)
		}

		target = cliConf.CurrentTarget()
	}

	switch cmd.(type) {
	case *serverCmd, *targetCmd:
	default:
		target = cliConf.CurrentTarget()
		if target == nil {
			bailWith("No doomsday server is currently targeted")
		}

		u, err := url.Parse(target.Address)
		if err != nil {
			bailWith("Could not parse target address as URL")
		}

		client = &doomsday.Client{
			URL:   *u,
			Token: target.Token,
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: target.SkipVerify,
					},
				},
			},
		}
	}

	err := cmd.Run()
	if err != nil {
		bailWith(err.Error())
	}

	err = cliConf.saveConfig(*configPath)
	if err != nil {
		bailWith("Could not save config: %s", err)
	}
}

func bailWith(f string, a ...interface{}) {
	ansi.Fprintf(os.Stderr, fmt.Sprintf("@R{%s}\n", f), a...)
	os.Exit(1)
}
