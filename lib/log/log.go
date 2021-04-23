package log

import (
	"context"

	"github.com/apex/log"
)

type (
	Fields map[string]interface{}

	Logger log.Interface

	LogLevel int
)

const (
	Debug LogLevel = iota
	Info
	Warning
	Error
)

func NewLogger(ctx context.Context, level LogLevel) Logger {
	var l log.Level

	switch level {
	case Debug:
		l = log.DebugLevel
	case Info:
		l = log.InfoLevel
	case Warning:
		l = log.WarnLevel
	case Error:
		l = log.ErrorLevel
	default:
		l = log.ErrorLevel
	}

	log.SetLevel(l)

	return log.FromContext(ctx)
}

func (f Fields) Fields() log.Fields {
	return log.Fields(f)
}
