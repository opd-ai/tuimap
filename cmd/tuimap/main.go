package main

import (
	"fmt"
	"os"

	"github.com/opd-ai/tuimap/internal/config"
	"github.com/opd-ai/tuimap/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Version information - set during build
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"

	// Flags
	noTUI bool
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
		if noTUI {
			// Headless mode - show help for now
			cmd.Help()
			return
		}

		// Start the TUI
		if err := tui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
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
		fmt.Printf("# Configuration loaded from: %s\n\n", cfg.ConfigPath)

		// Pretty print configuration as YAML
		output, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
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
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "run in headless mode without TUI")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
