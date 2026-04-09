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

// Banner attempts to grab the service banner
func (n *NetcatTool) Banner(ctx context.Context, host string, port int) (string, error)
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
```

> **Note:** The Dig tool also supports MX, TXT, NS, CNAME, and PTR record lookups through the `Execute` method with the appropriate record type argument (e.g., `dig example.com MX`).

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
func NewTengoEngine(maxTime time.Duration, maxMemoryMB int) *TengoEngine
```

Creates a new scripting engine.

**Parameters:**
- `maxTime`: Maximum script execution time (default: 30s)
- `maxMemoryMB`: Maximum memory usage in megabytes (default: 50)

**Methods:**
```go
// Run executes a script from string
func (e *TengoEngine) Run(ctx context.Context, source string) error

// LoadFile loads and executes a script from file
func (e *TengoEngine) LoadFile(ctx context.Context, path string) error

// SetAPIBridge sets the API bridge for network and device operations
func (e *TengoEngine) SetAPIBridge(api *APIBridge)

// Stop stops all running scripts
func (e *TengoEngine) Stop()

// IsRunning returns whether a script is currently running
func (e *TengoEngine) IsRunning() bool
```

### Script APIs

Functions exposed to Tengo scripts:

**Network:**
- `scan()` - Run network scan
- `ping(host)` - Ping a host
- `portScan(host, ports)` - Scan specific ports
- `resolve(hostname)` - DNS resolution

**Devices:**
- `getDevices()` - Get all devices
- `alert(level, message)` - Generate alert

**Storage:**
- `set(key, value)` - Store value
- `get(key)` - Get value

**Utility:**
- `print(args...)` - Print output
- `println(args...)` - Print output with newline

---

## NAT Package

**Import**: `github.com/opd-ai/tuimap/internal/nat`

The NAT package provides NAT detection and traversal.

### Functions

#### NewClient

```go
func NewClient(stunServers ...string) *Client
```

Creates a new NAT client. If no STUN servers are provided, uses default servers
(stun.l.google.com:19302, stun1.l.google.com:19302, stun.cloudflare.com:3478).

**Methods:**
```go
// Discover finds NAT devices and determines NAT type
func (c *Client) Discover(ctx context.Context) (*Info, error)

// GetExternalIP returns the public IP address via STUN
func (c *Client) GetExternalIP(ctx context.Context) (net.IP, error)

// AddPortMapping creates a port forwarding rule
func (c *Client) AddPortMapping(ctx context.Context, internal, external int, proto Protocol, desc string, lifetime time.Duration) (*PortMapping, error)

// RemovePortMapping removes a port forwarding rule
func (c *Client) RemovePortMapping(ctx context.Context, external int, proto Protocol) error

// ListMappings returns all active port mappings
func (c *Client) ListMappings(ctx context.Context) ([]PortMapping, error)
```

#### Info

```go
type Info struct {
    ExternalIP   net.IP        // External/public IP address
    InternalIP   net.IP        // Internal/private IP address
    GatewayIP    net.IP        // Gateway IP
    Type         Type          // NAT type (none, full_cone, restricted_cone, port_restricted, symmetric, unknown)
    UPnPEnabled  bool          // UPnP available
    NATPMPEnable bool          // NAT-PMP available
    Latency      time.Duration // Detection latency
}
```

> **Known Limitation**: NAT detection (type detection, public IP via STUN, UPnP/NAT-PMP discovery) is fully functional. Port mapping operations are not yet functionally implemented; internally, `addMappingUPnP` and `addMappingNATPMP` return `ErrNATUnsupported`, which causes `AddPortMapping` to return `ErrPortMapFailed`. `RemovePortMapping` is a no-op that returns `nil`.

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

#### InitConfig

```go
func InitConfig() error
```

Creates a default configuration file.

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
    Scripting ScriptingConfig
    Storage   StorageConfig
    Logging   LoggingConfig
}
```

---

## TUI Package

**Import**: `github.com/opd-ai/tuimap/internal/tui`

The TUI package provides the terminal user interface.

### Functions

#### NewModel

```go
func NewModel() Model
func NewModelWithOrchestrator(orch *scanner.Orchestrator, subnet string) Model
func NewModelWithOrchestratorAndStorage(orch *scanner.Orchestrator, subnet string, storage *tracker.Storage) Model
```

Creates a new TUI model. Use the variant that matches your setup.

**Run functions:**
```go
// Run starts the TUI with no pre-configured scanner
func Run() error

// RunWithOrchestrator starts the TUI with a scanner orchestrator
func RunWithOrchestrator(orch *scanner.Orchestrator, subnet string) error

// RunWithOrchestratorAndStorage starts the TUI with scanner and storage
func RunWithOrchestratorAndStorage(orch *scanner.Orchestrator, subnet string, storage *tracker.Storage) error
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

### Device

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

### Scanner Interface

```go
type Scanner interface {
    Scan(ctx context.Context, subnet string) (*ScanResult, error)
    ScanWithOptions(ctx context.Context, opts ScanOptions) (*ScanResult, error)
}
```

### Tracker Interface

```go
type Tracker interface {
    Update(devices []Device) error
    GetDevices() []Device
    GetDevice(ip string) (Device, error)
    GetAlerts() []Alert
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
