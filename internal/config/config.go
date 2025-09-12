package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

const (
	// DefaultSMTPPassword is the default password that SMTP clients must use.
	DefaultSMTPPassword = "smoggmos"
)

// Config stores all configuration for the application.
type Config struct {
	// LogLevel: Set the detail level for logs. Options: "Disabled", "Minimal", "Verbose".
	LogLevel string `mapstructure:"LogLevel"`
	// LogPath: Path to the log file. Platform-specific defaults are used if empty.
	LogPath string `mapstructure:"LogPath"`
	// GoogleCredentialsPath: Absolute path to the credentials.json file downloaded from Google Cloud.
	GoogleCredentialsPath string `mapstructure:"GoogleCredentialsPath"`
	// GoogleTokenPath: Path to store the generated OAuth2 token.
	GoogleTokenPath string `mapstructure:"GoogleTokenPath"`
	// SMTPUser: The username that SMTP clients must use to authenticate.
	SMTPUser string `mapstructure:"SMTPUser"`
	// SMTPPassword: The password that SMTP clients must use.
	SMTPPassword string `mapstructure:"SMTPPassword"`
	// SMTPPort: The TCP port for the SMTP server to listen on.
	SMTPPort int `mapstructure:"SMTPPort"`
	// MessageSizeLimitMB: The maximum email size (in Megabytes) to accept.
	MessageSizeLimitMB int `mapstructure:"MessageSizeLimitMB"`
	// AllowedSubnets: A list of allowed client IP addresses or CIDR subnets.
	AllowedSubnets []string `mapstructure:"AllowedSubnets"`
	// ReadTimeout: The maximum duration in seconds for reading the entire request.
	ReadTimeout int `mapstructure:"ReadTimeout"`
	// WriteTimeout: The maximum duration in seconds for writing the response.
	WriteTimeout int `mapstructure:"WriteTimeout"`
	// MaxRecipients: The maximum number of recipients for a single email.
	MaxRecipients int `mapstructure:"MaxRecipients"`
	// AllowInsecureAuth: Allow insecure authentication methods.
	AllowInsecureAuth bool `mapstructure:"AllowInsecureAuth"`
}

// getDefaultTokenPath returns the default path for the token file.
func getDefaultTokenPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "smog", "token.json"), nil
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	// Add platform-specific default search paths.
	switch runtime.GOOS {
	case "windows":
		viper.AddConfigPath(filepath.Join(os.Getenv("ProgramData"), "smog"))
	case "linux":
		viper.AddConfigPath("/etc/smog/")
		viper.AddConfigPath("/var/lib/smog/")
	case "darwin":
		viper.AddConfigPath("/Library/Application Support/smog/")
	}
	viper.AddConfigPath(".") // Always search in the current directory.

	viper.SetConfigName("smog")
	viper.SetConfigType("toml")

	viper.AutomaticEnv()
	// Bind specific environment variables to config keys. This is more explicit
	// and handles cases where the key name doesn't directly map to the env var name
	// (e.g., LogLevel -> LOG_LEVEL).
	viper.BindEnv("LogLevel", "LOG_LEVEL")
	viper.BindEnv("SMTPPort", "SMTP_PORT")
	viper.BindEnv("AllowedSubnets", "ALLOWED_SUBNETS")

	if path != "" {
		viper.SetConfigFile(path) // Use specific config file path if provided.
	}

	err = viper.ReadInConfig()
	if err != nil {
		// If the config file is not found, we can proceed with defaults,
		// but we will validate required fields later.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, fmt.Errorf("error reading config file: %w", err)
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// --- Validation and Defaulting ---

	// If GoogleTokenPath is not set, provide a platform-specific default.
	if config.GoogleTokenPath == "" {
		config.GoogleTokenPath, err = getDefaultTokenPath()
		if err != nil {
			return config, fmt.Errorf("failed to determine default token path: %w", err)
		}
	}

	// GoogleCredentialsPath is mandatory.
	if config.GoogleCredentialsPath == "" {
		return config, fmt.Errorf("mandatory configuration field 'GoogleCredentialsPath' is not set")
	}

	// Set defaults for new fields if they are not set
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 10
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 10
	}
	if config.MaxRecipients <= 0 {
		config.MaxRecipients = 50
	}

	// If AllowInsecureAuth is not set, default it to true for consistency
	// with the default configuration files.
	if !viper.IsSet("AllowInsecureAuth") {
		config.AllowInsecureAuth = true
	}

	return config, nil
}

// defaultConfigDirOverride is used for testing to override the default config directory.
var defaultConfigDirOverride string

// getDefaultConfigDir returns the platform-specific default directory for the config file.
func getDefaultConfigDir() string {
	if defaultConfigDirOverride != "" {
		return defaultConfigDirOverride
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramData"), "smog")
	case "linux":
		return "/etc/smog" // System-wide configuration
	case "darwin":
		return "/Library/Application Support/smog"
	default:
		// Fallback for other systems (e.g., BSD)
		home, err := os.UserHomeDir()
		if err != nil {
			return "." // Fallback to current directory if home is not available
		}
		return filepath.Join(home, ".config", "smog")
	}
}

// Create creates a default config file in the platform-specific default location.
func Create(logger *slog.Logger) error {
	configDir := getDefaultConfigDir()
	configFile := filepath.Join(configDir, "smog.toml")

	// Create the directory if it doesn't exist.
	// This is important for first-time setup.
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("failed to create config directory", "path", configDir, "err", err)
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	if _, err := os.Stat(configFile); err == nil {
		logger.Warn("config file already exists", "path", configFile)
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	logger.Info("writing default config file", "path", configFile)
	return os.WriteFile(configFile, []byte(defaultConfig), 0644)
}
