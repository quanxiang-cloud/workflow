package logger

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	logFormatJson = "json"
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

func NewLogger(logLevel string) log.Logger {
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

	switch logLevel {
	case logLevelDebug:
		logger = level.NewFilter(logger, level.AllowDebug())
	case logLevelInfo:
		logger = level.NewFilter(logger, level.AllowInfo())
	case logLevelWarn:
		logger = level.NewFilter(logger, level.AllowWarn())
	case logLevelError:
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	logger = log.With(logger, "ts", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)

	return logger
}
