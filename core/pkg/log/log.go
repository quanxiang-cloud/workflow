package log

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"os"
)

const (
	LogFormatJson = "json"
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

func NewLogger(logLevel string) log.Logger {
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

	switch logLevel {
	case LogLevelDebug:
		logger = level.NewFilter(logger, level.AllowDebug())
	case LogLevelInfo:
		logger = level.NewFilter(logger, level.AllowInfo())
	case LogLevelWarn:
		logger = level.NewFilter(logger, level.AllowWarn())
	case LogLevelError:
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	logger = log.With(logger, "ts", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)

	return logger
}
