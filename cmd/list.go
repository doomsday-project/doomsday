package main

import (
	"fmt"
	"os"
	"strings"
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

	fmt.Println("")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Common Name", "Expires In", "Path"})
	for _, result := range results {
		expiresIn := time.Until(time.Unix(result.NotAfter, 0))

		expStr := ansi.Sprintf("@R{EXPIRED}")
		if expiresIn > 0 {
			expStr = formatDuration(expiresIn)
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

func formatDuration(dur time.Duration) string {
	retSlice := []string{}
	if dur >= time.Hour*24*365 {
		retSlice = append(retSlice, ansi.Sprintf("%dy", dur/(time.Hour*24*365)))
	}

	if dur >= time.Hour*24 {
		retSlice = append(retSlice, ansi.Sprintf("%dd", int64((dur.Hours())/24)%365))
	}

	if dur >= time.Hour {
		retSlice = append(retSlice, ansi.Sprintf("%dh", int64(dur.Hours())%24))
	}

	retSlice = append(retSlice, ansi.Sprintf("%dm", int64(dur.Minutes())%60))
	return strings.Join(retSlice, " ")
}
