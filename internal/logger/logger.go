package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

func Init(serviceName string) {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	var logLevel zerolog.Level
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}).
		With().
		Str("service", serviceName).
		Timestamp().
		Logger()
}

func WithJobID(jobID string) *zerolog.Logger {
	l := Logger.With().Str("job_id", jobID).Logger()
	return &l
}

func WithCorrelationID(correlationID string) *zerolog.Logger {
	l := Logger.With().Str("correlation_id", correlationID).Logger()
	return &l
}

