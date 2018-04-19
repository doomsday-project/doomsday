package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
)

type targetsCmd struct {
}

func (t *targetsCmd) Run() error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Address", "Skip Verify"})
	for _, target := range cliConf.Targets {
		name := target.Name
		if target.Name == cliConf.Current {
			name = ansi.Sprintf("@G{%s}", target.Name)
		}

		skipVerify := fmt.Sprintf("%t", target.SkipVerify)
		if target.SkipVerify {
			skipVerify = ansi.Sprintf("@R{%s}", skipVerify)
		}

		table.Append([]string{name, target.Address, skipVerify})
	}

	table.SetBorder(false)
	fmt.Println("")
	table.Render()
	fmt.Println("")
	return nil
}
