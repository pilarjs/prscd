// Package util is a collection of utility functions.
package util

import (
	"context"
	"log/slog"
	"os"
)

type logLevelType int

const (
	// DEBUG level
	DEBUG logLevelType = iota
	// INFO level
	INFO
	// ERROR level
	ERROR
)

type plog struct {
	logLevel logLevelType
	outlog   *slog.Logger
	errlog   *slog.Logger
}

// Info prints log to Stdout.
func (l *plog) Info(format string, a ...any) {
	if l.logLevel > INFO {
		return
	}
	// _, _ = fmt.Fprintf(os.Stdout, format+"\r\n", a...)
	l.outlog.Info(format, a...)
}

// Inspect prints log to stdout, but will not add a newline
func (l *plog) Inspect(format string, a ...any) {
	if l.logLevel > INFO {
		return
	}
	// _, _ = fmt.Fprintf(os.Stdout, "\r\033[K"+format+"\r", a...)
	// l.outlog.Info(format, a...)
	l.outlog.Log(context.Background(), -5, format, a...)
}

// Error prints log to stderr.
func (l *plog) Error(format string, a ...any) {
	// if l.logLevel > DEBUG {
	// 	_, _ = fmt.Fprintf(os.Stderr, format+"\r\n", a...)
	// } else {
	// 	_, _ = fmt.Fprintf(os.Stderr, "\033[31m"+format+"\033[0m\r\n", a...)
	// }
	l.errlog.Error(format, a...)
}

// Debug log to stdout with colors.
func (l *plog) Debug(format string, a ...any) {
	if l.logLevel > DEBUG {
		return
	}
	// l.Info("\033[36m"+format+"\033[0m", a...)
	l.outlog.Debug(format, a...)
}

// Fatal prints log to stderr and exit.
func (l *plog) Fatal(err error) {
	// l.Error("FATAL:%s", err)
	l.errlog.Error("[FATAL]", "err", err)
	os.Exit(1)
}

// SetLogLevel set log level.
func (l *plog) SetLogLevel(lvl logLevelType) {
	l.logLevel = lvl
}

// Log is a global logger
var Log *plog

func init() {
	Log = &plog{
		logLevel: INFO,
		outlog:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
		errlog:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}
