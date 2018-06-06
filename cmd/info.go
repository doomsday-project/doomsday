package main

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

type infoCmd struct{}

func (i *infoCmd) Run() error {
	info, err := client.Info()
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"VERSION", info.Version})
	table.SetAutoFormatHeaders(false)
	table.SetHeaderColor(tablewriter.Color(tablewriter.FgMagentaColor, tablewriter.Bold), tablewriter.Color(tablewriter.BgBlackColor))
	table.Append([]string{"AUTH METHOD", string(info.AuthType)})
	table.SetColumnColor(tablewriter.Color(tablewriter.FgMagentaColor, tablewriter.Bold), tablewriter.Color(tablewriter.BgBlackColor))
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)
	table.Render()

	return nil
}
