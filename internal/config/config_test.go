package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreate(t *testing.T) {
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
	if err := Create(); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify the config file was created
	var configFile string
	switch runtime.GOOS {
	case "windows":
		configFile = filepath.Join(tmpDir, "smog", "smog.conf")
	case "linux":
		configFile = "/etc/smog/smog.conf"
	case "darwin":
		configFile = "/Library/Application Support/smog/smog.conf"
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("config file was not created: %s", configFile)
	}

	// Clean up the created file
	os.Remove(configFile)
}
