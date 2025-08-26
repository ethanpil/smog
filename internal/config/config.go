package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	viper.AddConfigPath("/etc/smog/")
	viper.AddConfigPath("$HOME/.config/smog")
	viper.AddConfigPath(".")
	viper.SetConfigName("smog")
	viper.SetConfigType("toml")

	viper.AutomaticEnv()

	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.SetConfigFile("smog.conf")
	}

	err = viper.ReadInConfig()
	if err != nil {
		// If the config file doesn't exist, we can continue with defaults.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	if config.GoogleTokenPath == "" {
		config.GoogleTokenPath, err = getDefaultTokenPath()
		if err != nil {
			return
		}
	}

	return
}

// Create creates a default config file.
func Create(logger *slog.Logger) error {
	configFile := "smog.conf"
	if _, err := os.Stat(configFile); err == nil {
		logger.Warn("config file already exists", "configFile", configFile)
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	logger.Debug("writing default config file", "configFile", configFile)
	return os.WriteFile(configFile, []byte(defaultConfig), 0644)
}
