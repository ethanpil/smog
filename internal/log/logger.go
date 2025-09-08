package log

import (
	"io"
	"log/slog"
	"os"
)

// Log levels from config
const (
	LevelDisabled = "Disabled"
	LevelMinimal  = "Minimal"
	LevelVerbose  = "Verbose"
)

// New creates a new logger based on the configuration.
// It supports structured logging (JSON), file output, and a verbose flag
// to enable simultaneous console output.
func New(level, path string, verbose bool) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case LevelDisabled:
		// Discard all logs.
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	case LevelMinimal:
		logLevel = slog.LevelInfo
	case LevelVerbose:
		logLevel = slog.LevelDebug
	default:
		logLevel = slog.LevelInfo // Default to Minimal.
	}

	var writers []io.Writer
	fileLogFailed := false

	// Add file writer if a path is specified.
	if path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			writers = append(writers, file)
		} else {
			fileLogFailed = true
			// Fallback to stderr for the error message, as the logger isn't fully set up.
			transientLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			transientLogger.Error("failed to open log file", "path", path, "err", err)
			transientLogger.Warn("logging will fallback to the console")
		}
	}

	// Add console writer for verbose mode, if no log path is set, or if file logging failed.
	if verbose || path == "" || fileLogFailed {
		writers = append(writers, os.Stdout)
	}

	// If there are no writers, discard logs. This can happen if LogLevel is not
	// "Disabled" but file logging fails and verbose is false.
	if len(writers) == 0 {
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	// Combine writers if necessary.
	multiWriter := io.MultiWriter(writers...)

	return slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: logLevel,
	}))
}
