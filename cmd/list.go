package main

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/duration"
)

type listCmd struct {
	Beyond *string
	Within *string
}

func (s *listCmd) Run() error {
	results, err := client.GetCache()
	if err != nil {
		return err
	}

	filter := doomsday.CacheItemFilter{}

	//Parse the durations
	if s.Beyond != nil && *s.Beyond != "" {
		dur, err := duration.Parse(*s.Beyond)
		if err != nil {
			return fmt.Errorf("When parsing beyond duration: %s", err)
		}

		filter.Beyond = &dur
	}

	if s.Within != nil && *s.Within != "" {
		dur, err := duration.Parse(*s.Within)
		if err != nil {
			return fmt.Errorf("When parsing within duration: %s", err)
		}

		filter.Within = &dur
	}

	results = results.Filter(filter)

	//Printing
	fmt.Println("")
	printList(results)

	return nil
}

func printList(items doomsday.CacheItems) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Common Name", "Expires In", "Path"})
	for _, result := range items {
		expiresIn := time.Until(time.Unix(result.NotAfter, 0))

		expStr := ansi.Sprintf("@R{EXPIRED}")
		if expiresIn > 0 {
			expStr = duration.Format(expiresIn)
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
}
