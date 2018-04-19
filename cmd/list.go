package main

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
)

type listCmd struct {
}

func (s *listCmd) Run() error {
	results, err := client.GetCache()
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Common Name", "Expires In", "Path"})
	for _, result := range results {
		expiresIn := time.Until(time.Unix(result.NotAfter, 0))

		expStr := ansi.Sprintf("@R{EXPIRED}")
		if expiresIn > 0 {
			expStr = fmt.Sprintf("%dh%dm", int(expiresIn.Hours()), int(expiresIn.Minutes()))
		}
		table.Append([]string{
			result.CommonName,
			expStr,
			result.Path,
		})
	}
	table.SetBorder(false)
	table.SetRowLine(true)
	table.Render()

	return nil
}
