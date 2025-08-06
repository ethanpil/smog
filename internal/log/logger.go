package log

import (
	"log/slog"
	"os"
)

// Log levels
const (
	LevelDebug = "Debug"
	LevelInfo  = "Info"
	LevelWarn  = "Warn"
	LevelError = "Error"
)

// New creates a new logger with the given log level.
func New(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelInfo:
		logLevel = slog.LevelInfo
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}
