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

func (l *Logger) WriteF(s string, a ...interface{}) {
	timestamp := time.Now().Format(time.Stamp)
	l.Write([]byte(fmt.Sprintf(fmt.Sprintf("%s: %s\n", timestamp, s), a...)))
}

func (l *Logger) Write(b []byte) (int, error) {
	return fmt.Fprintf(l.f, "%s", b)
}
