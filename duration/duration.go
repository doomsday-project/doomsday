package duration

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/starkandwayne/goutils/ansi"
)

//Format takes a duration and outputs a string representing that duration
func Format(dur time.Duration) string {
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

//Parse takes a duration string in the form of 1y2d3h4m and if it is properly
// formatted, the represented duration is returned. If it could not be parsed,
// an error is returned. Spaces are ignored. It is okay for any of the types
// of time to be omitted or in any particular order.
func Parse(f string) (dur time.Duration, err error) {
	var tokens []string
	tokens, err = tokenize(f)
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

func tokenize(f string) (ret []string, err error) {
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
