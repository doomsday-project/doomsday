package ansi

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/mattn/go-isatty"
)

var (
	colors = map[string]string{
		"k": "00;30", // black
		"K": "01;30", // black (BOLD)

		"r": "00;31", // red
		"R": "01;31", // red (BOLD)

		"g": "00;32", // green
		"G": "01;32", // green (BOLD)

		"y": "00;33", // yellow
		"Y": "01;33", // yellow (BOLD)

		"b": "00;34", // blue
		"B": "01;34", // blue (BOLD)

		"m": "00;35", // magenta
		"M": "01;35", // magenta (BOLD)
		"p": "00;35", // magenta
		"P": "01;35", // magenta (BOLD)

		"c": "00;36", // cyan
		"C": "01;36", // cyan (BOLD)

		"w": "00;37", // white
		"W": "01;37", // white (BOLD)
	}

	ansiTagRe = regexp.MustCompile(`^@[kKrRgGyYbBmMpPcCwW*]{$`)
	ansiRawRe = regexp.MustCompile("^\033\\[(0?[01];)?3[0-7]m")
)

var colorable = isatty.IsTerminal(os.Stdout.Fd())

func Color(c bool) {
	colorable = c
}

type colorStack []string

func (c *colorStack) push(s string) {
	*c = append(*c, s)
}

func (c *colorStack) pop() {
	*c = (*c)[:len(*c)-1]
}

func (c colorStack) topColor() []byte {
	if !colorable {
		return []byte("")
	}

	if len(c) == 0 {
		return []byte("\033[00m")
	}

	color := (c[len(c)-1])
	if colorCode, found := colors[color]; found {
		return []byte(fmt.Sprintf("\033[%sm", colorCode))
	}

	return []byte(color)
}

func colorize(s string) string {
	stack := colorStack{}
	ret := []byte{}

	var toSkip int
	for i := 0; i < len(s); i += toSkip {
		toSkip = 1

		if len(s)-i >= 3 && ansiTagRe.MatchString(s[i:i+3]) {
			// if @X{
			colorCode := s[i+1 : i+2]
			stack.push(colorCode)
			if colorable {
				ret = append(ret, []byte(fmt.Sprintf("\033[%sm", colors[colorCode]))...)
			}
			toSkip = 3

		} else if len(stack) > 0 && s[i] == '}' {
			stack.pop()
			ret = append(ret, stack.topColor()...)

		} else {
			ret = append(ret, s[i])
		}
	}
	return string(ret)
}

func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(colorize(format), a...)
}

func Fprintf(out io.Writer, format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(out, colorize(format), a...)
}

func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(colorize(format), a...)
}

func Errorf(format string, a ...interface{}) error {
	return fmt.Errorf(colorize(format), a...)
}
