package bcp

import (
	"log"
	"os"
)

type Logger interface {
	Info(string)
}

type stdLogger struct {
	*log.Logger
}

func (l stdLogger) Info(msg string) {
	l.Logger.Print(msg)
}

var logger Logger = stdLogger{log.New(os.Stderr, "", log.LstdFlags)}

// `SetLogger()` sets a logger for the package.
func SetLogger(l Logger) {
	logger = l
}
