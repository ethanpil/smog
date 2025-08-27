package config

import (
	"log/slog"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestLoadConfig(t *testing.T) {
	t.Run("ValidConfigFile", func(t *testing.T) {
		content := `
LogLevel = "Verbose"
LogPath = "/var/log/smog.log"
GoogleCredentialsPath = "/etc/smog/credentials.json"
GoogleTokenPath = "/etc/smog/token.json"
SMTPUser = "testuser"
SMTPPassword = "testpassword"
SMTPPort = 2526
MessageSizeLimitMB = 20
AllowedSubnets = ["192.168.1.0/24", "10.0.0.1"]
ReadTimeout = 20
WriteTimeout = 20
MaxRecipients = 100
AllowInsecureAuth = false
`
		tmpfile, err := os.CreateTemp("", "smog.conf")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		config, err := LoadConfig(tmpfile.Name())
		assert.NoError(t, err)

		expected := Config{
			LogLevel:              "Verbose",
			LogPath:               "/var/log/smog.log",
			GoogleCredentialsPath: "/etc/smog/credentials.json",
			GoogleTokenPath:       "/etc/smog/token.json",
			SMTPUser:              "testuser",
			SMTPPassword:          "testpassword",
			SMTPPort:              2526,
			MessageSizeLimitMB:    20,
			AllowedSubnets:        []string{"192.168.1.0/24", "10.0.0.1"},
			ReadTimeout:           20,
			WriteTimeout:          20,
			MaxRecipients:         100,
			AllowInsecureAuth:     false,
		}

		assert.Equal(t, expected, config)
	})

	t.Run("DefaultsForMissingValues", func(t *testing.T) {
		content := `
GoogleCredentialsPath = "/etc/smog/credentials.json"
`
		tmpfile, err := os.CreateTemp("", "smog.conf")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		config, err := LoadConfig(tmpfile.Name())
		assert.NoError(t, err)

		// Assert that default values are set for the new fields
		assert.Equal(t, 10, config.ReadTimeout)
		assert.Equal(t, 10, config.WriteTimeout)
		assert.Equal(t, 50, config.MaxRecipients)
		// The default for AllowInsecureAuth is the zero value, which is false.
		// The default config files set it to true, but if the value is not present at all,
		// it will be false. The validation logic I added only sets defaults for numeric values <= 0.
		// This is acceptable.
		assert.Equal(t, false, config.AllowInsecureAuth)
	})

	t.Run("NonExistentConfigFile", func(t *testing.T) {
		_, err := LoadConfig("non-existent-config-file.toml")
		assert.Error(t, err)
	})

	t.Run("MalformedConfigFile", func(t *testing.T) {
		content := `this is not valid toml`
		tmpfile, err := os.CreateTemp("", "smog.conf")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		_, err = LoadConfig(tmpfile.Name())
		assert.Error(t, err)
	})

	t.Run("OverrideWithEnvVars", func(t *testing.T) {
		// Set environment variables that should override file content.
		t.Setenv("LOG_LEVEL", "TestLevel")
		t.Setenv("SMTP_PORT", "9999")
		t.Setenv("ALLOWED_SUBNETS", "1.1.1.1,2.2.2.2")

		// Create a config file with different values.
		content := `
GoogleCredentialsPath = "/tmp/credentials.json"
LogLevel = "FileLevel"
SMTPPort = 1234
AllowedSubnets = ["3.3.3.3"]
`
		tmpfile, err := os.CreateTemp("", "smog.conf")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.WriteString(content)
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)

		config, err := LoadConfig(tmpfile.Name())
		assert.NoError(t, err)

		// Assert that environment variables took precedence.
		assert.Equal(t, "TestLevel", config.LogLevel)
		assert.Equal(t, 9999, config.SMTPPort)
		// Viper can automatically split comma-separated environment variables into slices.
		assert.Equal(t, []string{"1.1.1.1", "2.2.2.2"}, config.AllowedSubnets)
	})
}
