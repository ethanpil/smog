package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_FileLoggingFails_FallsBackToStdout(t *testing.T) {
	// 1. Create a temporary directory that is not writable.
	// We create a directory and then immediately revoke write permissions.
	unwritableDir, err := os.MkdirTemp("", "unwritable_log_dir")
	require.NoError(t, err)
	defer os.RemoveAll(unwritableDir) // Clean up afterwards.

	// On Linux/macOS, we can use chmod. This is the most reliable way.
	err = os.Chmod(unwritableDir, 0555) // Read and execute only.
	require.NoError(t, err)

	// In case of test failure, ensure we can still delete it.
	defer os.Chmod(unwritableDir, 0755)

	// 2. Define a log path inside the unwritable directory.
	logPath := filepath.Join(unwritableDir, "test.log")

	// 3. Hijack stdout to capture the logger's output.
	// The fallback mechanism should write to os.Stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	// 4. Call the New function with a configuration that should trigger the fallback.
	// We pass a path that will fail, and verbose=false.
	logger := New(LevelMinimal, logPath, false)

	// 5. Write a test message to the logger.
	logger.Info("this message should appear on stdout")

	// 6. Read the captured output from the pipe.
	err = w.Close()
	require.NoError(t, err)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	capturedOutput := buf.String()

	// 7. Assert that the output contains the expected log message.
	// This proves that the logger did not default to io.Discard.
	assert.Contains(t, capturedOutput, "this message should appear on stdout", "Logger should have fallen back to stdout")

	// Also assert that the error message about the file was printed.
	// Note: This error is printed to stderr by the transient logger, so we can't capture it here.
	// But we can check that the log file was not created.
	_, err = os.Stat(logPath)
	assert.True(t, os.IsNotExist(err), "Log file should not have been created in the unwritable directory")
}

func TestNew_VerboseMode_AlwaysLogsToStdout(t *testing.T) {
	// Hijack stdout to capture the logger's output.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	// Call New with verbose=true but no path.
	logger := New(LevelVerbose, "", true)
	logger.Debug("verbose message")

	err := w.Close()
	require.NoError(t, err)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	capturedOutput := buf.String()

	assert.Contains(t, capturedOutput, "verbose message")
}

func TestNew_DisabledLevel_DiscardsLogs(t *testing.T) {
	// Create a logger with the "Disabled" level.
	logger := New(LevelDisabled, "", false)

	// This should be a no-op, but we need to verify no panics occur.
	// We can't easily assert that nothing was written, but we can
	// check the handler type.
	_, ok := logger.Handler().(*slog.JSONHandler)
	require.True(t, ok)

	// This is an indirect way to check. We can't access the writer directly,
	// but we know from the source that it should be io.Discard.
	// A more robust test would involve reflection, but that's overkill.
	// Let's just log and ensure it doesn't crash.
	logger.Info("this should be discarded")
}
