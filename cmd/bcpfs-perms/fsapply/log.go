package fsapply

import (
	"log"
	"os"
)

type Logger interface {
	Debug(string)
	Info(string)
	Panic(string)
}

type stdLogger struct {
	*log.Logger
}

func (l stdLogger) Debug(msg string) {
	// The outside must set a logger to enable debug messages.
}

func (l stdLogger) Info(msg string) {
	l.Logger.Print(msg)
}

func (l stdLogger) Panic(msg string) {
	l.Logger.Panic(msg)
}

var logger Logger = stdLogger{log.New(os.Stderr, "", log.LstdFlags)}

// `SetLogger()` sets a logger for the package.
func SetLogger(l Logger) {
	logger = l
}
