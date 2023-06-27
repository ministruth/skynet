package log

import (
	"io"
	"os"
	"strings"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type F = map[string]interface{}

// New wraps logrus.
func New() *logrus.Logger {
	return logrus.StandardLogger()
}

// WrapEntry wraps error compatible to logrus entry.
func WrapEntry(entry *logrus.Entry, err error) *logrus.Entry {
	if err == nil {
		return entry
	}
	text := tracerr.Sprint(err)
	traceText := strings.Split(text, "\n")
	if len(traceText) > 1 {
		return entry.WithField("debug", traceText[1:]).WithField("error", err.Error())
	}
	return entry.WithField("debug", nil).WithField("error", err.Error())
}

// NewEntry wraps error compatible to logrus.
func NewEntry(err error) *logrus.Entry {
	return WrapEntry(logrus.NewEntry(New()), err)
}

// SetJSONFormat sets log format to JSON.
func SetJSONFormat() {
	logrus.SetFormatter(new(logrus.JSONFormatter))
}

// SetTextFormat sets log format to Text.
func SetTextFormat() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// ShowStack appends call stack to log.
// This operation cannot be undo.
func ShowStack() {
	logrus.AddHook(logrus_stack.StandardHook())
}

// SetOutput sets log output.
// If multiple writer provided, write to all of them.
// If no writer provided, use default (warning/error/fatal/panic to stderr, others to stdout).
func SetOutput(out ...io.Writer) {
	var cnt = len(out)
	if cnt > 1 {
		mw := io.MultiWriter(out...)
		logrus.SetOutput(mw)
	} else if cnt == 1 {
		logrus.SetOutput(out[0])
	} else {
		logrus.AddHook(&Hook{
			Writer: os.Stderr,
			LogLevels: []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.WarnLevel,
			},
		})
		logrus.AddHook(&Hook{
			Writer: os.Stdout,
			LogLevels: []logrus.Level{
				logrus.InfoLevel,
				logrus.DebugLevel,
			},
		})
	}
}
