package main

import (
	"fmt"
	"os"

	"github.com/ethanpil/smog/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "smog",
	Short: "smog is a simple smtp relay for gmail",
	Long:  `A fast and simple smtp relay for gmail that can be configured with a single command.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do nothing
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the smtp relay server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("serve command called")
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "authenticates with gmail",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auth command called")
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
		if err := config.Create(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(createCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
