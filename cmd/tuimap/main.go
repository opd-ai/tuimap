package main

import (
	"fmt"
	"os"

	"github.com/opd-ai/tuimap/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Version information - set during build
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "tuimap",
	Short: "TuiMap - Terminal-based network diagnostic and mapping tool",
	Long: `TuiMap is a terminal-based network diagnostic and mapping tool built in Go,
designed for real-time network analysis with an emphasis on speed and accuracy
in NAT environments.

Features:
  - Fast network scanning (<10s for /24 networks)
  - Real-time device tracking and alerts
  - Integrated network tools (netcat, telnet, traceroute, dig, whois)
  - Extensible scripting engine (d5/tengo)
  - Modern TUI interface with multiple views`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action: show help
		cmd.Help()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("TuiMap version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration file created successfully")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration loaded from: %s\n", cfg.ConfigPath)
		// TODO: Pretty print configuration
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.config/tuimap/config.yaml)")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug mode")
	rootCmd.PersistentFlags().StringP("interface", "i", "", "network interface to use")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
