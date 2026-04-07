package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opd-ai/tuimap/internal/config"
	"github.com/opd-ai/tuimap/internal/scanner"
	"github.com/opd-ai/tuimap/internal/tracker"
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

		// Get interface from flags
		ifaceName, _ := cmd.Flags().GetString("interface")

		// Create orchestrator with default scanners
		orch, err := scanner.CreateDefaultOrchestrator(ifaceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create scanner: %v\n", err)
		}

		// Detect subnet to scan
		subnet, err := scanner.DetectSubnet()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not detect subnet: %v\n", err)
			subnet = "" // TUI will show error when scan is attempted
		}

		// Create storage for device persistence
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not get home directory: %v\n", err)
			homeDir = "."
		}
		dbPath := filepath.Join(homeDir, ".local", "share", "tuimap", "tuimap.db")
		storage, err := tracker.NewStorage(dbPath, 30*24*time.Hour) // 30 day retention
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create storage: %v\n", err)
		}
		defer func() {
			if storage != nil {
				storage.Close()
			}
		}()

		// Start the TUI with orchestrator and storage
		if err := tui.RunWithOrchestratorAndStorage(orch, subnet, storage); err != nil {
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

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan network for devices (headless mode)",
	Long: `Scan the network for devices without starting the TUI.
Results can be output as JSON or text format.

Examples:
  tuimap scan                          # Scan auto-detected subnet
  tuimap scan --subnet 192.168.1.0/24  # Scan specific subnet
  tuimap scan --output json            # Output as JSON`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		subnet, _ := cmd.Flags().GetString("subnet")
		ifaceName, _ := cmd.Flags().GetString("interface")
		outputFormat, _ := cmd.Flags().GetString("output")
		timeoutSec, _ := cmd.Flags().GetInt("timeout")

		// Auto-detect subnet if not specified
		if subnet == "" {
			var err error
			subnet, err = scanner.DetectSubnet()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting subnet: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please specify --subnet manually\n")
				os.Exit(1)
			}
		}

		// Create orchestrator
		orch, err := scanner.CreateDefaultOrchestrator(ifaceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating scanner: %v\n", err)
			os.Exit(1)
		}

		// Run scan
		timeout := time.Duration(timeoutSec) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		fmt.Fprintf(os.Stderr, "Scanning %s...\n", subnet)
		result, err := orch.Scan(ctx, subnet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
			os.Exit(1)
		}

		// Output results
		if outputFormat == "json" {
			outputScanJSON(result)
		} else {
			outputScanText(result)
		}
	},
}

// outputScanJSON outputs scan results as JSON.
func outputScanJSON(result *scanner.ScanResult) {
	type deviceJSON struct {
		IP       string `json:"ip"`
		MAC      string `json:"mac,omitempty"`
		Hostname string `json:"hostname,omitempty"`
		Vendor   string `json:"vendor,omitempty"`
		Status   string `json:"status"`
		Ports    []int  `json:"ports,omitempty"`
	}

	devices := make([]deviceJSON, len(result.Devices))
	for i, d := range result.Devices {
		mac := ""
		if d.MAC != nil {
			mac = d.MAC.String()
		}
		devices[i] = deviceJSON{
			IP:       d.IP.String(),
			MAC:      mac,
			Hostname: d.Hostname,
			Vendor:   d.Vendor,
			Status:   string(d.Status),
			Ports:    d.Ports,
		}
	}

	output, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
}

// outputScanText outputs scan results as formatted text.
func outputScanText(result *scanner.ScanResult) {
	fmt.Printf("\nScan completed in %v\n", result.ScanTime.Round(time.Millisecond))
	fmt.Printf("Found %d devices:\n\n", len(result.Devices))

	for _, d := range result.Devices {
		mac := "unknown"
		if d.MAC != nil {
			mac = d.MAC.String()
		}
		fmt.Printf("  IP: %-15s  MAC: %-17s  Status: %s\n", d.IP, mac, d.Status)
		if d.Hostname != "" {
			fmt.Printf("      Hostname: %s\n", d.Hostname)
		}
		if d.Vendor != "" {
			fmt.Printf("      Vendor: %s\n", d.Vendor)
		}
		if len(d.Ports) > 0 {
			fmt.Printf("      Ports: %v\n", d.Ports)
		}
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(scanCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)

	// Scan command flags
	scanCmd.Flags().StringP("subnet", "s", "", "subnet to scan (e.g., 192.168.1.0/24)")
	scanCmd.Flags().StringP("output", "o", "text", "output format: text or json")
	scanCmd.Flags().IntP("timeout", "t", 15, "scan timeout in seconds")

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
