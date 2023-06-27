package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

type Hook struct {
	Writer    io.Writer
	LogLevels []logrus.Level
}

func (hook *Hook) Fire(entry *logrus.Entry) error {
	// reformat
	var fmt logrus.Formatter
	switch entry.Logger.Formatter.(type) {
	case *logrus.TextFormatter: // bug in determine TTY, must reinit here
		fmt = &logrus.TextFormatter{
			FullTimestamp: true,
		}
	default:
		fmt = entry.Logger.Formatter
	}
	entry.Logger.SetOutput(hook.Writer)
	line, err := fmt.Format(entry)
	entry.Logger.SetOutput(io.Discard)
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write(line)
	return err
}

func (hook *Hook) Levels() []logrus.Level {
	return hook.LogLevels
}
