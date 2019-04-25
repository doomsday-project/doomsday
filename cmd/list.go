package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/doomsday-project/doomsday/duration"
	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
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
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetReflowDuringAutoWrap(false)
	table.SetHeader([]string{"Common Name", "Expiry", "Path"})
	for _, result := range items {
		expiresIn := time.Until(time.Unix(result.NotAfter, 0))

		expStr := ansi.Sprintf("@R{EXPIRED}")
		if expiresIn > 0 {
			expStr = duration.Format(expiresIn)
		}
		table.Append([]string{
			result.CommonName,
			expStr,
			genPathStr(result),
		})
	}
	table.Render()
}

func genPathStr(item doomsday.CacheItem) string {
	fmtPaths := []string{}
	for i := 0; i < len(item.Paths); i++ {
		backendStr := item.Paths[i].Backend + "->"
		if i != 0 && item.Paths[i].Backend == item.Paths[i-1].Backend {
			b := make([]byte, len(backendStr))
			for j := range b {
				b[j] = 0x20
			}
			backendStr = string(b)
		}
		fmtPaths = append(fmtPaths, fmt.Sprintf("%s%s", backendStr, item.Paths[i].Location))
	}

	ret := strings.Join(fmtPaths, "\n")
	return ret
}
