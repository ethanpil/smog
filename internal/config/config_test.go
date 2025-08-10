package config

import (
	"log/slog"
	"os"
	"runtime"
	"testing"
)

func TestCreate(t *testing.T) {
	// Clean up any old config file before the test
	os.Remove("smog.conf")

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "smog-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the appropriate environment variable for the OS
	switch runtime.GOOS {
	case "windows":
		os.Setenv("ProgramData", tmpDir)
	case "linux":
		// The config path is hardcoded for linux
	case "darwin":
		// The config path is hardcoded for darwin
	}

	// Call the Create function
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := Create(logger); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the config file was created
	if _, err := os.Stat("smog.conf"); os.IsNotExist(err) {
		t.Errorf("config file was not created: smog.conf")
	}

	// Clean up the created file
	os.Remove("smog.conf")
}
