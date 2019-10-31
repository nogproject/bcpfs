/*

Package `zap` wraps Zap logging.

Zap has been chosen after a quick review of the logging solutions listed on
Awesome Go.  Zap was among the top 5 on GitHub.  Its performance seemed
impressive.  Its API is similar to the stdlib `log` package.

*/
package zap

import (
	"go.uber.org/zap"
)

// `Logger` is a Zap logger and also provides methods `LevelS(string)` that can
// be used in interfaces without Zap dependency.  The `S` is uppercase to avoid
// confusion with plural `s`.
type Logger struct {
	*zap.Logger
}

func (l *Logger) InfoS(msg string) {
	l.Info(msg)
}

func (l *Logger) WarnS(msg string) {
	l.Warn(msg)
}

func (l *Logger) ErrorS(msg string) {
	l.Error(msg)
}

func (l *Logger) FatalS(msg string) {
	l.Fatal(msg)
}

func NewProduction() (*Logger, error) {
	l, err := zap.NewProduction(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}
	return &Logger{l}, nil
}

func NewDevelopment() (*Logger, error) {
	l, err := zap.NewDevelopment(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}
	return &Logger{l}, nil
}
