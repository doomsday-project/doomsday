package logger

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	f *os.File
}

func NewLogger(f *os.File) *Logger {
	return &Logger{f: f}
}

func (l *Logger) Write(s string, a ...interface{}) {
	timestamp := time.Now().Format(time.Stamp)
	fmt.Fprintf(l.f, fmt.Sprintf("%s: %s\n", timestamp, s), a...)
}
