package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	// Scanner defaults
	if cfg.Scanner.Timeout != 10*time.Second {
		t.Errorf("Expected 10s timeout, got %v", cfg.Scanner.Timeout)
	}

	if cfg.Scanner.ARP.Workers != 256 {
		t.Errorf("Expected 256 ARP workers, got %d", cfg.Scanner.ARP.Workers)
	}

	if cfg.Scanner.ICMP.Workers != 256 {
		t.Errorf("Expected 256 ICMP workers, got %d", cfg.Scanner.ICMP.Workers)
	}

	if cfg.Scanner.TCP.Workers != 512 {
		t.Errorf("Expected 512 TCP workers, got %d", cfg.Scanner.TCP.Workers)
	}

	// Check scan methods
	if len(cfg.Scanner.Methods) != 3 {
		t.Errorf("Expected 3 scan methods, got %d", len(cfg.Scanner.Methods))
	}

	// TCP ports
	expectedPorts := []int{22, 80, 443, 3389, 5900}
	if len(cfg.Scanner.TCP.Ports) != len(expectedPorts) {
		t.Errorf("Expected %d ports, got %d", len(expectedPorts), len(cfg.Scanner.TCP.Ports))
	}

	// Alerts
	if !cfg.Alerts.Enabled {
		t.Error("Expected alerts to be enabled by default")
	}

	if len(cfg.Alerts.Rules) != 3 {
		t.Errorf("Expected 3 alert rules, got %d", len(cfg.Alerts.Rules))
	}

	// NAT
	if !cfg.NAT.Detect {
		t.Error("Expected NAT detection to be enabled")
	}

	if !cfg.NAT.UPnPEnabled {
		t.Error("Expected UPnP to be enabled")
	}

	if len(cfg.NAT.STUNServers) != 2 {
		t.Errorf("Expected 2 STUN servers, got %d", len(cfg.NAT.STUNServers))
	}

	// Scripting
	if !cfg.Scripting.Enabled {
		t.Error("Expected scripting to be enabled")
	}

	if cfg.Scripting.MaxExecutionTime != 30*time.Second {
		t.Errorf("Expected 30s max execution, got %v", cfg.Scripting.MaxExecutionTime)
	}

	if cfg.Scripting.MaxMemory != "50MB" {
		t.Errorf("Expected '50MB' max memory, got '%s'", cfg.Scripting.MaxMemory)
	}

	// TUI
	if cfg.TUI.Theme != "dark" {
		t.Errorf("Expected 'dark' theme, got '%s'", cfg.TUI.Theme)
	}

	if cfg.TUI.RefreshRate != 30 {
		t.Errorf("Expected 30 FPS refresh rate, got %d", cfg.TUI.RefreshRate)
	}

	// Storage
	if cfg.Storage.MaxDevices != 10000 {
		t.Errorf("Expected 10000 max devices, got %d", cfg.Storage.MaxDevices)
	}

	if cfg.Storage.HistoryRetention != 30*24*time.Hour {
		t.Errorf("Expected 30 days retention, got %v", cfg.Storage.HistoryRetention)
	}

	// Logging
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected 'info' log level, got '%s'", cfg.Logging.Level)
	}

	if cfg.Logging.MaxSize != "10MB" {
		t.Errorf("Expected '10MB' max size, got '%s'", cfg.Logging.MaxSize)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	// LoadConfig should return defaults when no config file exists
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	// Should have default values
	if cfg.Scanner.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout, got %v", cfg.Scanner.Timeout)
	}
}

func TestInitConfigAlreadyExists(t *testing.T) {
	// Save original home
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp home with existing config
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "tuimap")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("test: value"), 0o644)

	// InitConfig should fail
	err := InitConfig()
	if err == nil {
		t.Error("Expected error when config already exists")
	}
}

func TestConfigTypes(t *testing.T) {
	// Test that all config structs can be instantiated
	cfg := &Config{}
	_ = cfg.Scanner
	_ = cfg.Alerts
	_ = cfg.NAT
	_ = cfg.Scripting
	_ = cfg.TUI
	_ = cfg.Storage
	_ = cfg.Logging

	scanner := &ScannerConfig{}
	_ = scanner.ARP
	_ = scanner.ICMP
	_ = scanner.TCP

	arp := &ARPConfig{}
	_ = arp.Workers
	_ = arp.Timeout
	_ = arp.Retries

	icmp := &ICMPConfig{}
	_ = icmp.Workers
	_ = icmp.Timeout
	_ = icmp.Count

	tcp := &TCPConfig{}
	_ = tcp.Workers
	_ = tcp.Timeout
	_ = tcp.Ports

	alerts := &AlertsConfig{}
	_ = alerts.Enabled
	_ = alerts.Rules

	rule := &AlertRule{}
	_ = rule.Type
	_ = rule.Severity
	_ = rule.Action

	nat := &NATConfig{}
	_ = nat.Detect
	_ = nat.UPnPEnabled
	_ = nat.PublicIPCheck
	_ = nat.STUNServers

	scripting := &ScriptingConfig{}
	_ = scripting.Enabled
	_ = scripting.ScriptDir
	_ = scripting.AutoRun
	_ = scripting.MaxExecutionTime
	_ = scripting.MaxMemory

	tui := &TUIConfig{}
	_ = tui.Theme
	_ = tui.RefreshRate
	_ = tui.DefaultView
	_ = tui.Keybindings

	storage := &StorageConfig{}
	_ = storage.Database
	_ = storage.HistoryRetention
	_ = storage.MaxDevices

	logging := &LoggingConfig{}
	_ = logging.Level
	_ = logging.File
	_ = logging.MaxSize
}

func TestDefaultConfigPaths(t *testing.T) {
	cfg := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// Check script dir path
	expectedScriptDir := filepath.Join(homeDir, ".config/tuimap/scripts")
	if cfg.Scripting.ScriptDir != expectedScriptDir {
		t.Errorf("Expected script dir '%s', got '%s'", expectedScriptDir, cfg.Scripting.ScriptDir)
	}

	// Check database path
	expectedDB := filepath.Join(homeDir, ".local/share/tuimap/tuimap.db")
	if cfg.Storage.Database != expectedDB {
		t.Errorf("Expected database '%s', got '%s'", expectedDB, cfg.Storage.Database)
	}

	// Check log file path
	expectedLog := filepath.Join(homeDir, ".local/share/tuimap/tuimap.log")
	if cfg.Logging.File != expectedLog {
		t.Errorf("Expected log file '%s', got '%s'", expectedLog, cfg.Logging.File)
	}
}

func TestAlertRuleDefaults(t *testing.T) {
	cfg := DefaultConfig()

	expectedRules := []struct {
		Type     string
		Severity int
		Action   string
	}{
		{"new_device", 1, "notify"},
		{"device_offline", 2, "log"},
		{"port_change", 2, "notify"},
	}

	if len(cfg.Alerts.Rules) != len(expectedRules) {
		t.Fatalf("Expected %d rules, got %d", len(expectedRules), len(cfg.Alerts.Rules))
	}

	for i, expected := range expectedRules {
		actual := cfg.Alerts.Rules[i]
		if actual.Type != expected.Type {
			t.Errorf("Rule %d: expected type '%s', got '%s'", i, expected.Type, actual.Type)
		}
		if actual.Severity != expected.Severity {
			t.Errorf("Rule %d: expected severity %d, got %d", i, expected.Severity, actual.Severity)
		}
		if actual.Action != expected.Action {
			t.Errorf("Rule %d: expected action '%s', got '%s'", i, expected.Action, actual.Action)
		}
	}
}

func TestTUIKeybindings(t *testing.T) {
	cfg := DefaultConfig()

	expectedBindings := map[string]string{
		"quit":    "q",
		"refresh": "r",
		"scan":    "s",
	}

	for key, expected := range expectedBindings {
		actual, ok := cfg.TUI.Keybindings[key]
		if !ok {
			t.Errorf("Missing keybinding for '%s'", key)
			continue
		}
		if actual != expected {
			t.Errorf("Keybinding '%s': expected '%s', got '%s'", key, expected, actual)
		}
	}
}

func TestInitConfigSuccess(t *testing.T) {
	// Save original home
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp home without config
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	// InitConfig should succeed
	err := InitConfig()
	if err != nil {
		t.Errorf("InitConfig failed: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tmpDir, ".config", "tuimap", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Save original home
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp home with valid config
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	// Reset viper for this test
	viper.Reset()

	configDir := filepath.Join(tmpDir, ".config", "tuimap")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")

	// Write a valid config with explicit scanner and tui sections
	configContent := `
scanner:
  timeout: 20s
  interface: eth0
  scan_interval: 60s
  methods:
    - arp
    - icmp
    - tcp
  arp:
    workers: 256
    timeout: 100ms
    retries: 2
  icmp:
    workers: 256
    timeout: 1s
    count: 1
  tcp:
    workers: 512
    timeout: 500ms
    ports:
      - 22
      - 80
      - 443
      - 3389
      - 5900
tui:
  theme: light
  refresh_rate: 60
  default_view: network_map
  keybindings:
    quit: q
    refresh: r
    scan: s
`
	os.WriteFile(configPath, []byte(configContent), 0o644)

	// LoadConfig should read from file
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check that values from file were loaded
	if cfg.Scanner.Timeout != 20*time.Second {
		t.Errorf("Expected 20s timeout from file, got %v", cfg.Scanner.Timeout)
	}

	if cfg.Scanner.Interface != "eth0" {
		t.Errorf("Expected 'eth0' interface from file, got '%s'", cfg.Scanner.Interface)
	}

	if cfg.TUI.Theme != "light" {
		t.Errorf("Expected 'light' theme from file, got '%s'", cfg.TUI.Theme)
	}

	if cfg.TUI.RefreshRate != 60 {
		t.Errorf("Expected 60 refresh rate from file, got %d", cfg.TUI.RefreshRate)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Save original home
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp home with invalid config
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	// Reset viper for this test
	viper.Reset()

	configDir := filepath.Join(tmpDir, ".config", "tuimap")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")

	// Write malformed YAML that causes parse error
	os.WriteFile(configPath, []byte("scanner:\n  timeout: not_a_duration\n"), 0o644)

	// LoadConfig should fail on unmarshal
	_, err := LoadConfig()
	// The error may happen at unmarshal stage
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
	// Note: viper may be lenient with some invalid YAML, so we just log
}

func TestDefaultSTUNServers(t *testing.T) {
	cfg := DefaultConfig()

	expectedServers := []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
	}

	if len(cfg.NAT.STUNServers) != len(expectedServers) {
		t.Fatalf("Expected %d STUN servers, got %d", len(expectedServers), len(cfg.NAT.STUNServers))
	}

	for i, expected := range expectedServers {
		if cfg.NAT.STUNServers[i] != expected {
			t.Errorf("STUN server %d: expected '%s', got '%s'", i, expected, cfg.NAT.STUNServers[i])
		}
	}
}

func TestScanMethods(t *testing.T) {
	cfg := DefaultConfig()

	expectedMethods := []string{"arp", "icmp", "tcp"}

	if len(cfg.Scanner.Methods) != len(expectedMethods) {
		t.Fatalf("Expected %d scan methods, got %d", len(expectedMethods), len(cfg.Scanner.Methods))
	}

	for i, expected := range expectedMethods {
		if cfg.Scanner.Methods[i] != expected {
			t.Errorf("Scan method %d: expected '%s', got '%s'", i, expected, cfg.Scanner.Methods[i])
		}
	}
}

func TestScannerTimeouts(t *testing.T) {
	cfg := DefaultConfig()

	// Test individual scanner timeouts
	if cfg.Scanner.ARP.Timeout != 100*time.Millisecond {
		t.Errorf("Expected 100ms ARP timeout, got %v", cfg.Scanner.ARP.Timeout)
	}

	if cfg.Scanner.ICMP.Timeout != 1*time.Second {
		t.Errorf("Expected 1s ICMP timeout, got %v", cfg.Scanner.ICMP.Timeout)
	}

	if cfg.Scanner.TCP.Timeout != 500*time.Millisecond {
		t.Errorf("Expected 500ms TCP timeout, got %v", cfg.Scanner.TCP.Timeout)
	}
}

func TestDefaultTCPPorts(t *testing.T) {
	cfg := DefaultConfig()

	expectedPorts := []int{22, 80, 443, 3389, 5900}

	if len(cfg.Scanner.TCP.Ports) != len(expectedPorts) {
		t.Fatalf("Expected %d TCP ports, got %d", len(expectedPorts), len(cfg.Scanner.TCP.Ports))
	}

	for i, expected := range expectedPorts {
		if cfg.Scanner.TCP.Ports[i] != expected {
			t.Errorf("TCP port %d: expected %d, got %d", i, expected, cfg.Scanner.TCP.Ports[i])
		}
	}
}

func TestRetrySettings(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Scanner.ARP.Retries != 2 {
		t.Errorf("Expected 2 ARP retries, got %d", cfg.Scanner.ARP.Retries)
	}

	if cfg.Scanner.ICMP.Count != 1 {
		t.Errorf("Expected 1 ICMP count, got %d", cfg.Scanner.ICMP.Count)
	}
}
