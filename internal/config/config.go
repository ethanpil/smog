package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/viper"
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
		return
	}

	err = viper.Unmarshal(&config)
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
