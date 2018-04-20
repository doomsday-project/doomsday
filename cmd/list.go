package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/thomasmmitchell/doomsday"
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
		dur, err := parseDuration(*s.Beyond)
		if err != nil {
			return fmt.Errorf("When parsing beyond duration: %s", err)
		}

		filter.Beyond = &dur
	}

	if s.Within != nil && *s.Within != "" {
		dur, err := parseDuration(*s.Within)
		if err != nil {
			return fmt.Errorf("When parsing within duration: %s", err)
		}

		filter.Within = &dur
	}

	results.Filter(filter)

	//Printing
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

func parseDuration(f string) (dur time.Duration, err error) {
	var tokens []string
	tokens, err = tokenizeDuration(f)
	if err != nil {
		return
	}

	mustParseInt := func(s string) time.Duration {
		ret, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			panic("Could not parse int (should have been caught in tokenizer)")
		}

		return time.Duration(ret)
	}

	curNum := time.Duration(0)
	for _, token := range tokens {
		switch token {
		case "y":
			dur += time.Hour * 24 * 365 * curNum
		case "d":
			dur += time.Hour * 24 * curNum
		case "h":
			dur += time.Hour * curNum
		case "m":
			dur += time.Minute * curNum
		default: //number
			curNum = mustParseInt(token)
		}
	}

	return
}

func tokenizeDuration(f string) (ret []string, err error) {
	var curNum []byte
	for _, c := range f {
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			curNum = append(curNum, byte(c))
		case 'y', 'd', 'h', 'm':
			if len(curNum) == 0 {
				err = fmt.Errorf("Unit specifier found without number")
				goto doneTokenizing
			}

			ret = append(ret, string(curNum))
			curNum = nil
			ret = append(ret, fmt.Sprintf("%c", c))
		case ' ':
		default:
			err = fmt.Errorf("Unrecognized token found")
			goto doneTokenizing
		}
	}
doneTokenizing:

	return
}
