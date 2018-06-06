package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

type infoCmd struct{}

func (i *infoCmd) Run() error {
	info, err := client.Info()
	if err != nil {
		return err
	}

	fmt.Println("")
	table := tablewriter.NewWriter(os.Stdout)

	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)

	table.SetHeader([]string{"VERSION", info.Version})
	table.Append([]string{"AUTH METHOD", string(info.AuthType)})

	table.SetHeaderColor(tablewriter.Color(tablewriter.FgMagentaColor, tablewriter.Bold), tablewriter.Color(tablewriter.BgBlackColor))
	table.SetColumnColor(tablewriter.Color(tablewriter.FgMagentaColor, tablewriter.Bold), tablewriter.Color(tablewriter.BgBlackColor))
	table.Render()
	fmt.Println("")

	return nil
}
