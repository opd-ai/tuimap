// Package api provides public APIs for external integration.
// This package exports interfaces and types for programmatic use of TuiMap.
package api

import (
	"context"
	"net"
	"time"
)

// Scanner is the public interface for network scanning.
type Scanner interface {
	// Scan performs a network scan on the given subnet.
	Scan(ctx context.Context, subnet string) (*ScanResult, error)

	// ScanWithOptions performs a scan with custom options.
	ScanWithOptions(ctx context.Context, opts ScanOptions) (*ScanResult, error)
}

// Tracker is the public interface for device tracking.
type Tracker interface {
	// Update updates the device registry with new scan results.
	Update(devices []Device) error

	// GetDevices returns all tracked devices.
	GetDevices() []Device

	// GetDevice returns a specific device by IP.
	GetDevice(ip string) (Device, error)

	// GetAlerts returns all pending alerts.
	GetAlerts() []Alert
}

// NetworkTool is the public interface for network diagnostic tools.
type NetworkTool interface {
	// Name returns the tool name.
	Name() string

	// Execute runs the tool with the given arguments.
	Execute(ctx context.Context, args []string) (<-chan string, error)

	// Validate validates the arguments.
	Validate(args []string) error
}

// ScriptEngine is the public interface for the scripting engine.
type ScriptEngine interface {
	// Run executes a script from a string.
	Run(ctx context.Context, script string) error

	// LoadFile loads and runs a script from a file.
	LoadFile(ctx context.Context, path string) error

	// Stop stops all running scripts.
	Stop()
}

// Device represents a discovered network device.
type Device struct {
	IP        net.IP                 `json:"ip"`
	MAC       net.HardwareAddr       `json:"mac,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	Vendor    string                 `json:"vendor,omitempty"`
	Ports     []int                  `json:"ports,omitempty"`
	LastSeen  time.Time              `json:"last_seen"`
	FirstSeen time.Time              `json:"first_seen"`
	Status    DeviceStatus           `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DeviceStatus represents the current status of a device.
type DeviceStatus string

const (
	// StatusOnline indicates the device is currently online.
	StatusOnline DeviceStatus = "online"
	// StatusOffline indicates the device is currently offline.
	StatusOffline DeviceStatus = "offline"
	// StatusNew indicates a newly discovered device.
	StatusNew DeviceStatus = "new"
	// StatusChanged indicates a device with changed configuration.
	StatusChanged DeviceStatus = "changed"
)

// ScanResult contains the results of a network scan.
type ScanResult struct {
	Devices     []Device      `json:"devices"`
	ScanTime    time.Duration `json:"scan_time"`
	Method      string        `json:"method"`
	NetworkInfo NetworkInfo   `json:"network_info"`
}

// NetworkInfo contains metadata about the scanned network.
type NetworkInfo struct {
	Subnet    string `json:"subnet"`
	Gateway   net.IP `json:"gateway,omitempty"`
	Interface string `json:"interface,omitempty"`
}

// ScanOptions configures scan behavior.
type ScanOptions struct {
	Subnet     string        `json:"subnet"`
	Methods    []string      `json:"methods,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	ARPWorkers int           `json:"arp_workers,omitempty"`
	TCPPorts   []int         `json:"tcp_ports,omitempty"`
}

// Alert represents a triggered alert.
type Alert struct {
	Type      AlertType `json:"type"`
	Device    Device    `json:"device"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Severity  int       `json:"severity"`
}

// AlertType represents different types of alerts.
type AlertType string

const (
	// AlertNewDevice is triggered when a new device is detected.
	AlertNewDevice AlertType = "new_device"
	// AlertDeviceOffline is triggered when a device goes offline.
	AlertDeviceOffline AlertType = "device_offline"
	// AlertPortChange is triggered when a device's ports change.
	AlertPortChange AlertType = "port_change"
	// AlertMACConflict is triggered when a MAC address conflict is detected.
	AlertMACConflict AlertType = "mac_conflict"
)

// Config represents the public configuration interface.
type Config interface {
	// GetScannerConfig returns scanner configuration.
	GetScannerConfig() ScannerConfig

	// GetAlertConfig returns alert configuration.
	GetAlertConfig() AlertConfig
}

// ScannerConfig holds scanner settings.
type ScannerConfig struct {
	Interface    string        `json:"interface,omitempty"`
	ScanInterval time.Duration `json:"scan_interval"`
	Timeout      time.Duration `json:"timeout"`
	Methods      []string      `json:"methods"`
}

// AlertConfig holds alert settings.
type AlertConfig struct {
	Enabled bool        `json:"enabled"`
	Rules   []AlertRule `json:"rules"`
}

// AlertRule defines an alert rule.
type AlertRule struct {
	Type     string `json:"type"`
	Severity int    `json:"severity"`
	Action   string `json:"action"`
}
