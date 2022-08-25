package log

import (
	"io"
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

// NewEntry wraps error compatible to logrus.
func NewEntry(err error) *logrus.Entry {
	text := tracerr.Sprint(err)
	traceText := strings.Split(text, "\n")
	if len(traceText) > 1 {
		return logrus.WithField("debug", traceText[1:]).WithField("error", err.Error())
	}
	return logrus.WithField("debug", nil).WithField("error", err.Error())
}

func MergeEntry(a *logrus.Entry, b *logrus.Entry) *logrus.Entry {
	return a.WithFields(b.Data)
}

// SetJSONFormat sets log format to JSON.
func SetJSONFormat() {
	logrus.SetFormatter(new(logrus.JSONFormatter))
}

// SetTextFormat sets log format to Text.
func SetTextFormat() {
	logrus.SetFormatter(new(logrus.TextFormatter))
}

// ShowStack appends call stack to log.
// This operation cannot be undo.
func ShowStack() {
	logrus.AddHook(logrus_stack.StandardHook())
}

// SetOutput sets log output.
// If multiple writer provided, write to all of them.
// If no writer provided, do nothing.
func SetOutput(out ...io.Writer) {
	var cnt = len(out)
	if cnt > 1 {
		mw := io.MultiWriter(out...)
		logrus.SetOutput(mw)
	} else if cnt == 1 {
		logrus.SetOutput(out[0])
	}
	// do nothing if no input.
}
