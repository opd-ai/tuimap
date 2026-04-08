# TuiMap API Documentation

This document provides comprehensive API documentation for TuiMap's internal packages and interfaces.

## Table of Contents

1. [Scanner Package](#scanner-package)
2. [Tracker Package](#tracker-package)
3. [Tools Package](#tools-package)
4. [Script Package](#script-package)
5. [NAT Package](#nat-package)
6. [Config Package](#config-package)
7. [TUI Package](#tui-package)
8. [Public API](#public-api)

---

## Scanner Package

**Import**: `github.com/opd-ai/tuimap/internal/scanner`

The scanner package provides network scanning functionality for device discovery.

### Types

#### Device

Represents a discovered network device.

```go
type Device struct {
    IP        net.IP                 // Device IP address
    MAC       net.HardwareAddr       // MAC address (may be nil)
    Hostname  string                 // Resolved hostname
    Vendor    string                 // Vendor from OUI database
    Ports     []int                  // Open ports discovered
    LastSeen  time.Time              // Last discovery time
    FirstSeen time.Time              // First discovery time
    Status    DeviceStatus           // Current status
    Metadata  map[string]interface{} // Additional metadata
}
```

#### DeviceStatus

```go
type DeviceStatus string

const (
    StatusOnline  DeviceStatus = "online"
    StatusOffline DeviceStatus = "offline"
    StatusNew     DeviceStatus = "new"
    StatusChanged DeviceStatus = "changed"
)
```

#### ScanResult

Contains results from a network scan.

```go
type ScanResult struct {
    Devices     []Device        // Discovered devices
    ScanTime    time.Duration   // Total scan duration
    Method      string          // Scan method used
    NetworkInfo NetworkMetadata // Network metadata
}
```

#### NetworkMetadata

```go
type NetworkMetadata struct {
    Subnet    string // CIDR notation
    Gateway   net.IP // Gateway IP
    Interface string // Network interface name
}
```

### Interfaces

#### Scanner

Common interface for all scanner implementations.

```go
type Scanner interface {
    // Scan performs a network scan on the given subnet
    Scan(ctx context.Context, subnet string) ([]Device, error)
    
    // Name returns the scanner name
    Name() string
}
```

### Functions

#### NewARPScanner

```go
func NewARPScanner(ifaceName string, workers int, timeout time.Duration, retries int) (*ARPScanner, error)
```

Creates a new ARP scanner. Requires root privileges.

**Parameters:**
- `ifaceName`: Network interface name (empty for auto-detect)
- `workers`: Number of concurrent workers (recommended: 256)
- `timeout`: Per-host timeout (recommended: 100ms)
- `retries`: Number of retries per host

#### NewICMPScanner

```go
func NewICMPScanner(workers int, timeout time.Duration, count int) *ICMPScanner
```

Creates a new ICMP scanner. Requires root privileges for privileged mode.

**Parameters:**
- `workers`: Number of concurrent workers (recommended: 256)
- `timeout`: Per-host timeout (recommended: 1s)
- `count`: Pings per host

#### NewTCPScanner

```go
func NewTCPScanner(workers int, timeout time.Duration, ports []int) *TCPScanner
```

Creates a new TCP scanner. No root required.

**Parameters:**
- `workers`: Number of concurrent workers (recommended: 512)
- `timeout`: Per-connection timeout (recommended: 500ms)
- `ports`: Ports to scan (nil for defaults: 22, 80, 443, 3389, 5900)

#### NewOrchestrator

```go
func NewOrchestrator(timeout time.Duration) *Orchestrator
```

Creates a scan orchestrator for coordinating multiple scanners.

#### CreateDefaultOrchestrator

```go
func CreateDefaultOrchestrator(ifaceName string) (*Orchestrator, error)
```

Creates an orchestrator with default scanner configuration.

### Multi-Subnet Scanning

#### DiscoverSubnets

```go
func DiscoverSubnets() ([]SubnetInfo, error)
```

Discovers all local subnets on the system.

#### ParseRoutingTable

```go
func ParseRoutingTable() ([]SubnetInfo, error)
```

Parses the system routing table to find subnets.

#### NewMultiSubnetScanner

```go
func NewMultiSubnetScanner(orch *Orchestrator) *MultiSubnetScanner
```

Creates a scanner that can handle multiple subnets.

---

## Tracker Package

**Import**: `github.com/opd-ai/tuimap/internal/tracker`

The tracker package provides device state management and alerting.

### Types

#### Alert

```go
type Alert struct {
    Type      AlertType     // Alert type
    Device    scanner.Device // Related device
    Timestamp time.Time     // When alert occurred
    Message   string        // Alert message
    Severity  int           // 1=low, 2=medium, 3=high
}
```

#### AlertType

```go
type AlertType string

const (
    AlertNewDevice    AlertType = "new_device"
    AlertDeviceOffline AlertType = "device_offline"
    AlertPortChange   AlertType = "port_change"
    AlertMACConflict  AlertType = "mac_conflict"
)
```

### Registry

Thread-safe device registry with alert generation.

#### NewRegistry

```go
func NewRegistry(offlineThreshold time.Duration) *Registry
```

Creates a new device registry.

**Parameters:**
- `offlineThreshold`: Duration after which devices are marked offline

#### Registry Methods

```go
// Update updates the registry with discovered devices
func (r *Registry) Update(devices []scanner.Device) error

// GetDevice returns a device by IP address
func (r *Registry) GetDevice(ip string) (scanner.Device, error)

// GetDevices returns all tracked devices
func (r *Registry) GetDevices() []scanner.Device

// Count returns total device count
func (r *Registry) Count() int

// OnlineCount returns online device count
func (r *Registry) OnlineCount() int

// Clear removes all devices
func (r *Registry) Clear()

// GetAlerts returns and clears pending alerts
func (r *Registry) GetAlerts() []Alert

// Export exports devices as JSON
func (r *Registry) Export() ([]byte, error)
```

### Storage

Persistent storage using bbolt database.

#### NewStorage

```go
func NewStorage(path string, retention time.Duration) (*Storage, error)
```

Creates a new storage instance.

**Parameters:**
- `path`: Database file path
- `retention`: History retention duration

#### Storage Methods

```go
// SaveDevice saves a device to storage
func (s *Storage) SaveDevice(device scanner.Device) error

// SaveDevices saves multiple devices
func (s *Storage) SaveDevices(devices []scanner.Device) error

// LoadDevices loads all devices from storage
func (s *Storage) LoadDevices() ([]scanner.Device, error)

// SaveAlert saves an alert to storage
func (s *Storage) SaveAlert(alert Alert) error

// LoadAlerts loads all alerts from storage
func (s *Storage) LoadAlerts() ([]Alert, error)

// Close closes the storage
func (s *Storage) Close() error
```

---

## Tools Package

**Import**: `github.com/opd-ai/tuimap/internal/tools`

The tools package provides network diagnostic tool implementations.

### Interface

#### NetworkTool

```go
type NetworkTool interface {
    // Name returns the tool name
    Name() string
    
    // Execute runs the tool and returns output channel
    Execute(ctx context.Context, args []string) (<-chan string, error)
    
    // Validate validates arguments
    Validate(args []string) error
}
```

### Netcat Tool

```go
func NewNetcatTool(timeout time.Duration) *NetcatTool
```

Creates a netcat (nc) tool for TCP/UDP connections.

**Methods:**
```go
// TCPConnect establishes a TCP connection
func (n *NetcatTool) TCPConnect(ctx context.Context, host string, port int) (bool, time.Duration, error)

// UDPSend sends UDP data
func (n *NetcatTool) UDPSend(ctx context.Context, host string, port int, data []byte) error
```

### Telnet Tool

```go
func NewTelnetTool(timeout time.Duration) *TelnetTool
```

Creates a telnet tool for protocol testing.

### Traceroute Tool

```go
func NewTracerouteTool(maxHops int, timeout time.Duration) *TracerouteTool
```

Creates a traceroute tool for path discovery.

**Types:**
```go
type Hop struct {
    IP       net.IP
    Hostname string
    RTT      time.Duration
    Reached  bool
    Timeout  bool
}
```

### Dig Tool

```go
func NewDigTool(timeout time.Duration, server string) *DigTool
```

Creates a DNS query tool.

**Methods:**
```go
// LookupIP performs A/AAAA record lookup
func (d *DigTool) LookupIP(ctx context.Context, hostname string) ([]net.IP, error)

// LookupMX performs MX record lookup
func (d *DigTool) LookupMX(ctx context.Context, hostname string) ([]*net.MX, error)

// LookupTXT performs TXT record lookup
func (d *DigTool) LookupTXT(ctx context.Context, hostname string) ([]string, error)

// LookupNS performs NS record lookup
func (d *DigTool) LookupNS(ctx context.Context, hostname string) ([]*net.NS, error)

// LookupCNAME performs CNAME record lookup
func (d *DigTool) LookupCNAME(ctx context.Context, hostname string) (string, error)
```

### Whois Tool

```go
func NewWhoisTool(timeout time.Duration) *WhoisTool
```

Creates a WHOIS lookup tool.

---

## Script Package

**Import**: `github.com/opd-ai/tuimap/internal/script`

The script package provides Tengo scripting engine integration.

### Engine

```go
func NewEngine(maxExecTime time.Duration, maxMemory int64) *Engine
```

Creates a new scripting engine.

**Parameters:**
- `maxExecTime`: Maximum script execution time (default: 30s)
- `maxMemory`: Maximum memory usage (default: 50MB)

**Methods:**
```go
// Run executes a script from string
func (e *Engine) Run(ctx context.Context, script string) (interface{}, error)

// RunFile executes a script from file
func (e *Engine) RunFile(ctx context.Context, path string) (interface{}, error)

// SetScanner sets the scanner for network operations
func (e *Engine) SetScanner(s *scanner.Orchestrator)

// SetRegistry sets the registry for device operations
func (e *Engine) SetRegistry(r *tracker.Registry)
```

### Script APIs

Functions exposed to Tengo scripts:

**Network:**
- `scan()` - Run network scan
- `ping(host)` - Ping a host
- `port_scan(host, ports)` - Scan specific ports
- `resolve(hostname)` - DNS resolution

**Devices:**
- `get_devices()` - Get all devices
- `get_device(ip)` - Get specific device
- `alert(type, message)` - Generate alert

**Storage:**
- `set(key, value)` - Store value
- `get(key)` - Get value
- `delete(key)` - Delete value

---

## NAT Package

**Import**: `github.com/opd-ai/tuimap/internal/nat`

The NAT package provides NAT detection and traversal.

### Functions

#### NewDetector

```go
func NewDetector(config *config.NATConfig) *Detector
```

Creates a NAT detector.

**Methods:**
```go
// Detect performs NAT detection
func (d *Detector) Detect(ctx context.Context) (*NATInfo, error)

// GetPublicIP returns public IP via STUN
func (d *Detector) GetPublicIP(ctx context.Context) (net.IP, error)

// GetGateway returns the default gateway
func (d *Detector) GetGateway() (net.IP, error)
```

#### NATInfo

```go
type NATInfo struct {
    BehindNAT   bool      // Whether behind NAT
    NATType     string    // NAT type (symmetric, full-cone, etc.)
    PublicIP    net.IP    // External IP address
    GatewayIP   net.IP    // Gateway IP
    UPnPEnabled bool      // UPnP available
    NATPMPEnabled bool    // NAT-PMP available
}
```

> **Known Limitation**: NAT detection (type detection, public IP via STUN, UPnP/NAT-PMP discovery) is fully functional. Port mapping operations are not yet functionally implemented; currently, `AddPortMapping` returns `ErrPortMapFailed`, and `RemovePortMapping` is a no-op that returns `nil`.

---

## Config Package

**Import**: `github.com/opd-ai/tuimap/internal/config`

The config package provides configuration management.

### Functions

#### LoadConfig

```go
func LoadConfig() (*Config, error)
```

Loads configuration from file and environment.

#### SaveConfig

```go
func SaveConfig(cfg *Config) error
```

Saves configuration to file.

#### DefaultConfig

```go
func DefaultConfig() *Config
```

Returns default configuration.

### Config Structure

```go
type Config struct {
    Scanner   ScannerConfig
    Alerts    AlertsConfig
    NAT       NATConfig
    TUI       TUIConfig
    Scripting ScriptConfig
    Storage   StorageConfig
    Logging   LogConfig
}
```

---

## TUI Package

**Import**: `github.com/opd-ai/tuimap/internal/tui`

The TUI package provides the terminal user interface.

### Functions

#### NewApp

```go
func NewApp(cfg *config.Config, scanner *scanner.Orchestrator, registry *tracker.Registry) *App
```

Creates a new TUI application.

**Methods:**
```go
// Run starts the TUI
func (a *App) Run() error
```

### Views

The TUI provides four main views:
- Network Map View (`1`)
- Device List View (`2`)
- Tools View (`3`)
- Script Console View (`4`)

---

## Public API

**Import**: `github.com/opd-ai/tuimap/pkg/api`

The public API package provides stable interfaces for external integration.

### Device Interface

```go
type Device interface {
    IP() net.IP
    MAC() net.HardwareAddr
    Hostname() string
    Vendor() string
    Ports() []int
    LastSeen() time.Time
    Status() string
}
```

### Scanner Interface

```go
type NetworkScanner interface {
    Scan(ctx context.Context, subnet string) ([]Device, error)
    ScanAll(ctx context.Context) ([]Device, error)
}
```

### Tracker Interface

```go
type DeviceTracker interface {
    GetDevice(ip string) (Device, error)
    GetDevices() []Device
    GetAlerts() []Alert
    Subscribe() <-chan Alert
}
```

---

## Error Handling

All packages follow consistent error handling patterns:

```go
// Check for specific errors
if errors.Is(err, ErrNotFound) {
    // Handle not found
}

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to scan: %w", err)
}
```

### Common Errors

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrTimeout      = errors.New("operation timed out")
    ErrPermission   = errors.New("permission denied")
    ErrInvalidInput = errors.New("invalid input")
)
```

---

## Concurrency Safety

All exported types are safe for concurrent use:

- `Registry`: Protected by `sync.RWMutex`
- `Storage`: Uses bbolt transactions
- `Orchestrator`: Runs scanners in goroutines with proper synchronization
- `Engine`: Each script runs in isolated context

---

## Example Usage

### Basic Scanning

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/opd-ai/tuimap/internal/scanner"
)

func main() {
    orch, _ := scanner.CreateDefaultOrchestrator("")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    result, err := orch.Scan(ctx, "192.168.1.0/24")
    if err != nil {
        panic(err)
    }
    
    for _, device := range result.Devices {
        fmt.Printf("%s - %s\n", device.IP, device.Hostname)
    }
}
```

### Device Tracking

```go
package main

import (
    "github.com/opd-ai/tuimap/internal/tracker"
    "github.com/opd-ai/tuimap/internal/scanner"
    "time"
)

func main() {
    registry := tracker.NewRegistry(5 * time.Minute)
    
    // After each scan
    devices := []scanner.Device{...}
    registry.Update(devices)
    
    // Check for alerts
    for _, alert := range registry.GetAlerts() {
        fmt.Printf("Alert: %s - %s\n", alert.Type, alert.Message)
    }
}
```
