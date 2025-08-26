package main

import (
	"fmt"
	"os"

	"github.com/ethanpil/smog/internal/app"
	"github.com/ethanpil/smog/internal/auth"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/log"
	"github.com/spf13/cobra"
)

// Global flags
var (
	configPath string
	verbose    bool
	silent     bool
)

var rootCmd = &cobra.Command{
	Use:   "smog",
	Short: "smog is a simple smtp relay for gmail",
	Long:  `A fast and simple smtp relay for gmail that can be configured with a single command.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no command is specified, default to the 'serve' command
		serveCmd.Run(cmd, args)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the smtp relay server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error: failed to load configuration: %v\n", err)
			os.Exit(1)
		}

		// Determine log level from config and flags
		logLevel := cfg.LogLevel
		if silent {
			logLevel = log.LevelDisabled
		}
		// The -v flag now controls verbosity at the logger level,
		// which also logs to console.
		logger := log.New(logLevel, cfg.LogPath, verbose)

		// --- Validation Checks (from AGENTS.md) ---

		// 1. Check for default password
		if cfg.SMTPPassword == config.DefaultSMTPPassword {
			logger.Error("security risk: the SMTP password is set to the default value", "password", config.DefaultSMTPPassword)
			logger.Error("please change SMTPPassword in your configuration file before running the server")
			os.Exit(1)
		}

		// 2. Check for authorization token
		token, err := auth.LoadToken(logger, &cfg)
		if err != nil {
			logger.Error("failed to load google api token", "err", err)
			os.Exit(1)
		}
		if token == nil {
			logger.Error("google api token not found or invalid")
			logger.Error("please run 'smog auth login' to authorize with google")
			os.Exit(1)
		}

		logger.Info("configuration and credentials validated successfully")

		if err := app.Run(&cfg, logger, nil); err != nil {
			logger.Error("failed to start server", "err", err)
			os.Exit(1)
		}
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "manages gmail authentication",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "authenticates with gmail",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error: failed to load configuration: %v\n", err)
			os.Exit(1)
		}
		// Auth command should be verbose by default to guide the user.
		logger := log.New(log.LevelVerbose, cfg.LogPath, true)
		if err := auth.Login(logger, &cfg); err != nil {
			logger.Error("failed to authenticate", "err", err)
			os.Exit(1)
		}
	},
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "revokes gmail authentication",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auth revoke command called")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "configures the smtp relay",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a default config file",
	Run: func(cmd *cobra.Command, args []string) {
		// Create command should always log to console.
		logger := log.New(log.LevelMinimal, "", true)
		if err := config.Create(logger); err != nil {
			logger.Error("failed to create config file", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging to console")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Disable all logging")

	// Add subcommands
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(revokeCmd)
	configCmd.AddCommand(createCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error, so we just need to exit
		os.Exit(1)
	}
}
