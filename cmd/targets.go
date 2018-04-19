package main

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
)

type targetsCmd struct {
}

func (t *targetsCmd) Run() error {
	toPrint := make([]string, 0, len(cliConf.Targets))
	for _, target := range cliConf.Targets {
		if target.Name == cliConf.Current {
			toPrint = append(toPrint, target.String())
		} else {
			toPrint = append(toPrint, ansi.Sprintf("@G{%s}", target.String()))
		}
	}

	fmt.Println(strings.Join(toPrint, "===\n"))
	return nil
}
