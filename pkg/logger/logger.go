package logger

import (
	"go.uber.org/zap"
)

// Logger wraps zap for structured logging
type Logger struct {
	*zap.SugaredLogger
}

// New creates a new logger with the specified level and format
func New(level, format string) *Logger {
	var zapConfig zap.Config

	if format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
	}
}
