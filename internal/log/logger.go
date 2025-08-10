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

// New creates a new logger with the given log level and path.
func New(level, path string) *slog.Logger {
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

	// Default to stdout
	w := os.Stdout
	if path != "" {
		// If a path is provided, try to open the file for writing.
		// The file is created if it does not exist.
		// New entries are appended to the file.
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			w = file
		}
		// If we can't open the file, we'll just log to stdout.
		// We could also log an error message to stdout here.
	}

	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: logLevel,
	}))
}
