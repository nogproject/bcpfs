package main

// Zap has been chosen for logging after a quick review of the logging
// solutions listed on Awesome Go.  Zap was among the top 5 on GitHub.  Its
// performance seemed impressive.  Its API is similar to the stdlib `log`
// package, so that dependency injection is easy; see package `fsapply`.

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/fsapply"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/fsck"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	*zap.Logger
}

func (l *zapLogger) Debug(msg string) {
	l.Logger.Debug(msg)
}

func (l *zapLogger) Info(msg string) {
	l.Logger.Info(msg)
}

func (l *zapLogger) Error(msg string) {
	l.Logger.Error(msg)
}

func (l *zapLogger) Panic(msg string) {
	l.Logger.Panic(msg)
}

func (l *zapLogger) Fatal(msg string) {
	l.Logger.Fatal(msg)
}

var logger *zapLogger = withInitImports(mustNewProductionLogger())

func InitDebugLogger() {
	logger = withInitImports(mustNewDebugLogger())
}

func withInitImports(l *zapLogger) *zapLogger {
	bcp.SetLogger(l)
	fsapply.SetLogger(l)
	fsck.SetLogger(l)
	return l
}

// Start from `NewDevelopmentConfig()`, since it seems more suitable for
// console output.  Disable `Development` and `Stacktrace` for all logging
// levels, since call stacks are too verbose and of little value.  Use
// `AddCallerSkip(1)` to hide our wrapping functions `Debug()` and so on.

func mustNewProductionLogger() *zapLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	cfg.Development = false
	cfg.DisableStacktrace = true
	l, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("failed to init Zap: %s", err))
	}
	return &zapLogger{l}
}

func mustNewDebugLogger() *zapLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Development = false
	cfg.DisableStacktrace = true
	l, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("failed to init Zap: %s", err))
	}
	return &zapLogger{l}
}
