// Package logger builds the application-wide structured logger.
package logger

import (
	"log/slog"
	"os"
)

// New builds a slog.Logger that writes JSON records to stdout. In
// development it logs at debug level; in staging/production it logs at
// info level to keep noise down.
func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
