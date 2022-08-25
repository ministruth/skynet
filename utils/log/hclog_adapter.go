package log

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/sirupsen/logrus"
)

type HCLogAdapter struct {
	Logger      logrus.FieldLogger
	PrependName string
	Args        []interface{}
}

func (h *HCLogAdapter) shouldEmit(level logrus.Level) bool {
	return h.Logger.WithFields(logrus.Fields{}).Level >= level
}

func (h *HCLogAdapter) CreateEntry(args []interface{}) *logrus.Entry {
	if len(args)%2 != 0 {
		args = append(args, "<unknown>")
	}

	fields := make(logrus.Fields)
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		v := args[i+1]
		fields[k] = v
	}

	return h.Logger.WithFields(fields)
}

func (h *HCLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		h.Trace(msg, args...)
	case hclog.Debug:
		h.Debug(msg, args...)
	case hclog.Info:
		h.Info(msg, args...)
	case hclog.Warn:
		h.Warn(msg, args...)
	case hclog.Error:
		h.Error(msg, args...)
	}
}

// Emit a message and key/value pairs at the TRACE level
func (h *HCLogAdapter) Trace(msg string, args ...interface{}) {
	h.CreateEntry(args).Trace(msg)
}

// Emit a message and key/value pairs at the DEBUG level
func (h *HCLogAdapter) Debug(msg string, args ...interface{}) {
	h.CreateEntry(args).Debug(msg)
}

// Emit a message and key/value pairs at the INFO level
func (h *HCLogAdapter) Info(msg string, args ...interface{}) {
	h.CreateEntry(args).Info(msg)
}

// Emit a message and key/value pairs at the WARN level
func (h *HCLogAdapter) Warn(msg string, args ...interface{}) {
	h.CreateEntry(args).Warn(msg)
}

// Emit a message and key/value pairs at the ERROR level
func (h *HCLogAdapter) Error(msg string, args ...interface{}) {
	h.CreateEntry(args).Error(msg)
}

// Indicate if TRACE logs would be emitted. This and the other Is* guards
// are used to elide expensive logging code based on the current level.
func (h *HCLogAdapter) IsTrace() bool {
	return h.shouldEmit(logrus.TraceLevel)
}

// Indicate if DEBUG logs would be emitted. This and the other Is* guards
func (h *HCLogAdapter) IsDebug() bool {
	return h.shouldEmit(logrus.DebugLevel)
}

// Indicate if INFO logs would be emitted. This and the other Is* guards
func (h *HCLogAdapter) IsInfo() bool {
	return h.shouldEmit(logrus.InfoLevel)
}

// Indicate if WARN logs would be emitted. This and the other Is* guards
func (h *HCLogAdapter) IsWarn() bool {
	return h.shouldEmit(logrus.WarnLevel)
}

// Indicate if ERROR logs would be emitted. This and the other Is* guards
func (h *HCLogAdapter) IsError() bool {
	return h.shouldEmit(logrus.ErrorLevel)
}

// ImpliedArgs returns With key/value pairs
func (h *HCLogAdapter) ImpliedArgs() []interface{} {
	return h.Args
}

func (h *HCLogAdapter) concatFields(b []interface{}) []interface{} {
	c := make([]interface{}, len(h.Args)+len(b))
	copy(c, h.Args)
	copy(c[len(h.Args):], b)
	return c
}

// Creates a sublogger that will always have the given key/value pairs
func (h *HCLogAdapter) With(args ...interface{}) hclog.Logger {
	e := h.CreateEntry(args)
	return &HCLogAdapter{
		Logger: e,
		Args:   h.concatFields(args),
	}
}

// Returns the Name of the logger
func (h *HCLogAdapter) Name() string { return h.PrependName }

// Create a logger that will prepend the name string on the front of all messages.
// If the logger already has a name, the new value will be appended to the current
// name. That way, a major subsystem can use this to decorate all it's own logs
// without losing context.
func (h *HCLogAdapter) Named(name string) hclog.Logger {
	var newName bytes.Buffer
	if h.PrependName != "" {
		newName.WriteString(h.PrependName)
		newName.WriteString(".")
	}
	newName.WriteString(name)

	return h.ResetNamed(newName.String())
}

// Create a logger that will prepend the name string on the front of all messages.
// This sets the name of the logger to the value directly, unlike Named which honor
// the current name as well.
func (h *HCLogAdapter) ResetNamed(name string) hclog.Logger {
	fields := []interface{}{"subsystem_name", name}
	e := h.CreateEntry(fields)
	return &HCLogAdapter{Logger: e, PrependName: name}
}

// Updates the level. This should affect all related loggers as well,
// unless they were created with IndependentLevels. If an
// implementation cannot update the level on the fly, it should no-op.
func (h *HCLogAdapter) SetLevel(level hclog.Level) {
	// we dont want to change level for hc_log
}

// Return a value that conforms to the stdlib log.Logger interface
func (h *HCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(h.Logger.WithFields(logrus.Fields{}).WriterLevel(logrus.InfoLevel), "", 0)
}

// Return a value that conforms to io.Writer, which can be passed into log.SetOutput()
func (h *HCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	var w io.Writer
	logger, ok := h.Logger.(*logrus.Logger)
	if ok {
		w = logger.Out
	}
	if w == nil {
		w = os.Stderr
	}
	return w
}
