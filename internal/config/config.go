package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	ConfigPath string
	Scanner    ScannerConfig   `mapstructure:"scanner"`
	Alerts     AlertsConfig    `mapstructure:"alerts"`
	NAT        NATConfig       `mapstructure:"nat"`
	Scripting  ScriptingConfig `mapstructure:"scripting"`
	TUI        TUIConfig       `mapstructure:"tui"`
	Storage    StorageConfig   `mapstructure:"storage"`
	Logging    LoggingConfig   `mapstructure:"logging"`
}

// ScannerConfig holds network scanning settings
type ScannerConfig struct {
	Interface    string        `mapstructure:"interface"`
	ScanInterval time.Duration `mapstructure:"scan_interval"`
	Timeout      time.Duration `mapstructure:"timeout"`
	Methods      []string      `mapstructure:"methods"`
	ARP          ARPConfig     `mapstructure:"arp"`
	ICMP         ICMPConfig    `mapstructure:"icmp"`
	TCP          TCPConfig     `mapstructure:"tcp"`
}

// ARPConfig holds ARP scanning settings
type ARPConfig struct {
	Workers int           `mapstructure:"workers"`
	Timeout time.Duration `mapstructure:"timeout"`
	Retries int           `mapstructure:"retries"`
}

// ICMPConfig holds ICMP scanning settings
type ICMPConfig struct {
	Workers int           `mapstructure:"workers"`
	Timeout time.Duration `mapstructure:"timeout"`
	Count   int           `mapstructure:"count"`
}

// TCPConfig holds TCP scanning settings
type TCPConfig struct {
	Workers int           `mapstructure:"workers"`
	Timeout time.Duration `mapstructure:"timeout"`
	Ports   []int         `mapstructure:"ports"`
}

// AlertsConfig holds alert settings
type AlertsConfig struct {
	Enabled bool        `mapstructure:"enabled"`
	Rules   []AlertRule `mapstructure:"rules"`
}

// AlertRule defines an alert rule
type AlertRule struct {
	Type     string `mapstructure:"type"`
	Severity int    `mapstructure:"severity"`
	Action   string `mapstructure:"action"`
}

// NATConfig holds NAT-related settings
type NATConfig struct {
	Detect        bool     `mapstructure:"detect"`
	UPnPEnabled   bool     `mapstructure:"upnp_enabled"`
	PublicIPCheck bool     `mapstructure:"public_ip_check"`
	STUNServers   []string `mapstructure:"stun_servers"`
}

// ScriptingConfig holds scripting engine settings
type ScriptingConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	ScriptDir        string        `mapstructure:"script_dir"`
	AutoRun          []string      `mapstructure:"auto_run"`
	MaxExecutionTime time.Duration `mapstructure:"max_execution_time"`
	MaxMemory        string        `mapstructure:"max_memory"`
}

// TUIConfig holds TUI settings
type TUIConfig struct {
	Theme       string            `mapstructure:"theme"`
	RefreshRate int               `mapstructure:"refresh_rate"`
	DefaultView string            `mapstructure:"default_view"`
	Keybindings map[string]string `mapstructure:"keybindings"`
}

// StorageConfig holds database settings
type StorageConfig struct {
	Database         string        `mapstructure:"database"`
	HistoryRetention time.Duration `mapstructure:"history_retention"`
	MaxDevices       int           `mapstructure:"max_devices"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	File    string `mapstructure:"file"`
	MaxSize string `mapstructure:"max_size"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Scanner: ScannerConfig{
			Interface:    "",
			ScanInterval: 60 * time.Second,
			Timeout:      10 * time.Second,
			Methods:      []string{"arp", "icmp", "tcp"},
			ARP: ARPConfig{
				Workers: 256,
				Timeout: 100 * time.Millisecond,
				Retries: 2,
			},
			ICMP: ICMPConfig{
				Workers: 256,
				Timeout: 1 * time.Second,
				Count:   1,
			},
			TCP: TCPConfig{
				Workers: 512,
				Timeout: 500 * time.Millisecond,
				Ports:   []int{22, 80, 443, 3389, 5900},
			},
		},
		Alerts: AlertsConfig{
			Enabled: true,
			Rules: []AlertRule{
				{Type: "new_device", Severity: 1, Action: "notify"},
				{Type: "device_offline", Severity: 2, Action: "log"},
				{Type: "port_change", Severity: 2, Action: "notify"},
			},
		},
		NAT: NATConfig{
			Detect:        true,
			UPnPEnabled:   true,
			PublicIPCheck: true,
			STUNServers: []string{
				"stun.l.google.com:19302",
				"stun1.l.google.com:19302",
			},
		},
		Scripting: ScriptingConfig{
			Enabled:          true,
			ScriptDir:        filepath.Join(homeDir, ".config/tuimap/scripts"),
			AutoRun:          []string{},
			MaxExecutionTime: 30 * time.Second,
			MaxMemory:        "50MB",
		},
		TUI: TUIConfig{
			Theme:       "dark",
			RefreshRate: 30,
			DefaultView: "network_map",
			Keybindings: map[string]string{
				"quit":    "q",
				"refresh": "r",
				"scan":    "s",
			},
		},
		Storage: StorageConfig{
			Database:         filepath.Join(homeDir, ".local/share/tuimap/tuimap.db"),
			HistoryRetention: 30 * 24 * time.Hour, // 30 days
			MaxDevices:       10000,
		},
		Logging: LoggingConfig{
			Level:   "info",
			File:    filepath.Join(homeDir, ".local/share/tuimap/tuimap.log"),
			MaxSize: "10MB",
		},
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config/tuimap/config.yaml")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(homeDir, ".config/tuimap"))
	viper.AddConfigPath(".")

	// Set defaults
	cfg := DefaultConfig()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// If config file doesn't exist, return defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cfg.ConfigPath = configPath
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal config
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.ConfigPath = viper.ConfigFileUsed()
	return cfg, nil
}

// InitConfig creates a default configuration file
func InitConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config/tuimap")
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config file
	cfg := DefaultConfig()

	// Write config using viper
	viper.SetConfigFile(configPath)

	// Set all values from default config
	viper.Set("scanner", cfg.Scanner)
	viper.Set("alerts", cfg.Alerts)
	viper.Set("nat", cfg.NAT)
	viper.Set("scripting", cfg.Scripting)
	viper.Set("tui", cfg.TUI)
	viper.Set("storage", cfg.Storage)
	viper.Set("logging", cfg.Logging)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
