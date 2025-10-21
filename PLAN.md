# TuiMap - Network Analysis TUI Implementation Plan

## 1. Architecture

### 1.1 System Overview
TuiMap is a terminal-based network diagnostic and mapping tool built in Go, designed for real-time network analysis with an emphasis on speed and accuracy in NAT environments.

```
┌─────────────────────────────────────────────────────────────┐
│                     TUI Layer (bubbletea)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Network  │  │ Device   │  │ Tool     │  │ Script   │   │
│  │ Map View │  │ List View│  │ View     │  │ Console  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Core Engine Layer                         │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │ Network Scanner  │  │ Device Tracker   │               │
│  │ - ARP Scanner    │  │ - State Manager  │               │
│  │ - ICMP Prober    │  │ - Alert Engine   │               │
│  │ - Port Scanner   │  │ - History DB     │               │
│  └──────────────────┘  └──────────────────┘               │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │ Network Tools    │  │ Script Engine    │               │
│  │ - netcat         │  │ - d5/tengo VM    │               │
│  │ - telnet         │  │ - API Bridge     │               │
│  │ - traceroute     │  │ - Event System   │               │
│  │ - dig            │  │                  │               │
│  │ - whois          │  │                  │               │
│  └──────────────────┘  └──────────────────┘               │
└─────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              System/Network Interface Layer                 │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Raw Socket │  │ System DNS │  │ OS Network │           │
│  │ Interface  │  │ Resolver   │  │ APIs       │           │
│  └────────────┘  └────────────┘  └────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Design

#### 1.2.1 Network Scanner
**Purpose:** Discover all devices on local network(s) in <10 seconds

**Strategy:**
- **Multi-method Parallel Scanning:**
  - ARP scan (fastest, layer 2, works on local subnet)
  - ICMP ping sweep (layer 3, crosses subnets)
  - Active TCP SYN scan on common ports (80, 443, 22)
  - Passive network traffic monitoring
  - NetBIOS/mDNS discovery for hostname resolution

**Key Design Points:**
- Worker pool pattern with configurable concurrency (default: 256 goroutines)
- Subnet detection and automatic CIDR calculation
- NAT traversal support via UPnP/NAT-PMP for gateway info
- Response aggregation to deduplicate multi-method discoveries
- Timeout per method: 3s (total budget: 9s + 1s processing)

**Data Structures:**
```go
type Device struct {
    IP          net.IP
    MAC         net.HardwareAddr
    Hostname    string
    Vendor      string
    Ports       []int
    LastSeen    time.Time
    FirstSeen   time.Time
    Status      DeviceStatus
    Metadata    map[string]interface{}
}

type ScanResult struct {
    Devices     []Device
    ScanTime    time.Duration
    Method      string
    NetworkInfo NetworkMetadata
}
```

#### 1.2.2 Device Tracker
**Purpose:** Maintain real-time device state and trigger alerts

**Features:**
- In-memory device registry with LRU cache
- Event-driven state changes (new, online, offline, changed)
- Alert conditions:
  - New device detected
  - Device went offline
  - Port changes
  - MAC address conflict/spoofing
- Persistent storage (SQLite for history)
- Export capabilities (JSON, CSV)

**Alert System:**
```go
type Alert struct {
    Type      AlertType
    Device    Device
    Timestamp time.Time
    Message   string
    Severity  int
}

type AlertRule struct {
    Condition func(Device) bool
    Action    func(Alert)
}
```

#### 1.2.3 Network Tools Integration
**Implementation approach:**
- Each tool as a separate module with common interface
- Execute as goroutines with context cancellation
- Stream output to TUI in real-time
- History of executed commands

**Tool Interfaces:**
```go
type NetworkTool interface {
    Name() string
    Execute(ctx context.Context, args []string) (<-chan string, error)
    Validate(args []string) error
}

// Implementations:
// - NetcatTool: TCP/UDP connection testing
// - TelnetTool: Protocol testing and banner grabbing
// - TracerouteTool: Path discovery with hop timing
// - DigTool: DNS query interface (A, AAAA, MX, TXT, etc.)
// - WhoisTool: Domain/IP registration lookup
```

#### 1.2.4 d5/tengo Scripting Engine
**Purpose:** User-extensible automation and custom network operations

**Integration:**
- Embedded tengo VM
- Exposed API for network operations
- Script hot-reloading
- Sandboxed execution with resource limits

**Available Script APIs:**
```go
// Core functions exposed to scripts
- scan(subnet string) []Device
- ping(ip string) bool
- portScan(ip string, ports []int) []int
- resolve(hostname string) string
- alert(message string, severity int)
- getData(key string) interface{}
- setData(key string, value interface{})
```

**Example Script:**
```tengo
// Auto-scan and alert on new devices
devices := scan("192.168.1.0/24")
for device in devices {
    if device.FirstSeen == device.LastSeen {
        alert("New device: " + device.IP, 1)
    }
}
```

#### 1.2.5 TUI Design
**Framework:** Bubble Tea (charmbracelet)

**Views:**
1. **Network Map View** (default)
   - Visual topology of discovered devices
   - Connection lines showing gateway relationships
   - Real-time status indicators
   - Navigation: arrow keys, vim bindings

2. **Device List View**
   - Sortable table of all devices
   - Columns: IP, MAC, Hostname, Vendor, Status, Last Seen
   - Quick filter/search
   - Details pane on selection

3. **Tool View**
   - Tab-based interface for each network tool
   - Input area for commands
   - Scrollable output area
   - Command history

4. **Script Console**
   - Script editor/loader
   - Execution controls
   - Output/error display
   - Variable inspector

**Keybindings:**
- `1-4`: Switch views
- `s`: Start scan
- `r`: Refresh
- `f`: Filter/search
- `t`: Open tool menu
- `q`: Quit
- `/`: Command mode
- `:`: Script mode

### 1.3 NAT Environment Optimization

**Challenges:**
- Limited visibility outside local subnet
- Gateway address detection
- Public IP discovery
- Port forwarding detection

**Solutions:**
1. **Local Network Focus:**
   - Primary scan on directly connected subnet
   - Use ARP for guaranteed local discovery
   - mDNS/LLMNR for cross-subnet discovery

2. **Gateway Integration:**
   - UPnP/NAT-PMP for router info
   - SNMP queries (if credentials available)
   - Parse routing tables for multi-subnet awareness

3. **Public IP Detection:**
   - Query STUN servers
   - Parse gateway UPnP external IP
   - Fallback to external HTTP services

4. **Smart Scanning:**
   - Detect if behind NAT via TTL analysis
   - Adjust scan strategy based on network topology
   - Focus resources on reachable addresses

## 2. Implementation Phases

### Phase 0: Project Setup (Week 1)
**Duration:** 5 days

**Tasks:**
- [x] Initialize Go module and repository structure
- [ ] Set up development environment
- [ ] Choose and integrate dependencies
- [ ] Create basic project structure
- [ ] Set up CI/CD pipeline

**Deliverables:**
```
tuimap/
├── cmd/
│   └── tuimap/
│       └── main.go
├── internal/
│   ├── scanner/
│   ├── tracker/
│   ├── tools/
│   ├── script/
│   └── tui/
├── pkg/
│   └── api/
├── scripts/
├── docs/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

**Dependencies:**
```go
// Core
- github.com/charmbracelet/bubbletea (TUI framework)
- github.com/charmbracelet/lipgloss (styling)
- github.com/d5/tengo/v2 (scripting)

// Networking
- github.com/google/gopacket (packet capture)
- github.com/jackpal/gateway (gateway detection)
- golang.org/x/net/icmp (ICMP)

// Storage
- github.com/mattn/go-sqlite3 (database)

// Utilities
- github.com/spf13/cobra (CLI)
- github.com/spf13/viper (config)
```

### Phase 1: Core Network Scanner (Week 2-3)
**Duration:** 10 days

**Milestones:**
1. **ARP Scanner** (Day 1-3)
   - Raw socket implementation
   - Subnet detection
   - MAC vendor lookup (OUI database)
   - Testing: verify <5s scan on /24 network

2. **ICMP Ping Scanner** (Day 4-5)
   - Concurrent ping sweep
   - Privilege handling (raw sockets)
   - Result aggregation
   - Testing: benchmark against fping

3. **Port Scanner** (Day 6-7)
   - TCP SYN scan for common ports
   - Service detection (banner grabbing)
   - Rate limiting to avoid detection
   - Testing: compare with nmap

4. **Integration & Optimization** (Day 8-10)
   - Parallel execution of all methods
   - Result deduplication and merging
   - Performance tuning for <10s total scan
   - Testing: full integration tests on various network sizes

**Success Criteria:**
- ✅ Detect 100% of active devices on /24 network in <10s
- ✅ Handle /16 networks gracefully (with longer timeout)
- ✅ Graceful degradation on permission issues
- ✅ Accurate device information (IP, MAC, hostname)

### Phase 2: Device Tracking System (Week 4)
**Duration:** 5 days

**Milestones:**
1. **State Management** (Day 1-2)
   - In-memory device registry
   - State change detection
   - Thread-safe operations
   - Testing: concurrent access, race conditions

2. **Alert Engine** (Day 3)
   - Alert rule system
   - Event dispatching
   - Notification formatting
   - Testing: alert triggering accuracy

3. **Persistence Layer** (Day 4-5)
   - SQLite schema design
   - History tracking
   - Query interface
   - Testing: data integrity, performance

**Success Criteria:**
- ✅ Real-time device status updates
- ✅ Alert latency <500ms
- ✅ Handle 1000+ device history
- ✅ No data loss on restart

### Phase 3: Network Tools (Week 5-6)
**Duration:** 10 days

**Milestones:**
1. **Tool Framework** (Day 1-2)
   - Common interface definition
   - Process management
   - Output streaming
   - Testing: framework reliability

2. **Tool Implementations** (Day 3-8)
   - Netcat (Day 3-4): TCP/UDP client/server
   - Telnet (Day 4): Protocol implementation
   - Traceroute (Day 5-6): Hop detection with timing
   - Dig (Day 7): DNS query types
   - Whois (Day 8): WHOIS protocol client
   - Testing: per-tool validation

3. **Integration** (Day 9-10)
   - Tool selection interface
   - Command history
   - Output persistence
   - Testing: end-to-end tool usage

**Success Criteria:**
- ✅ All 5 tools functional
- ✅ Real-time output streaming
- ✅ Proper error handling
- ✅ Command history persistence

### Phase 4: Scripting Engine (Week 7)
**Duration:** 5 days

**Milestones:**
1. **Tengo Integration** (Day 1-2)
   - VM initialization
   - Standard library setup
   - Resource limits
   - Testing: VM stability

2. **API Bridge** (Day 3-4)
   - Expose network functions to scripts
   - Event system for scripts
   - Data storage interface
   - Testing: API coverage, security

3. **Script Management** (Day 5)
   - Script loading/reloading
   - Error reporting
   - Example scripts
   - Testing: script execution, error handling

**Success Criteria:**
- ✅ Scripts can perform all core operations
- ✅ Sandboxed execution (no system damage)
- ✅ Hot reload without restart
- ✅ Clear error messages

### Phase 5: TUI Implementation (Week 8-9)
**Duration:** 10 days

**Milestones:**
1. **Core TUI Framework** (Day 1-2)
   - Bubble Tea setup
   - View switching logic
   - Keybinding system
   - Testing: navigation flow

2. **Network Map View** (Day 3-4)
   - Device layout algorithm
   - Topology rendering
   - Real-time updates
   - Testing: visual correctness, performance

3. **Device List View** (Day 5-6)
   - Table component
   - Sorting and filtering
   - Detail panel
   - Testing: data display accuracy

4. **Tool View** (Day 7-8)
   - Tab interface
   - Input/output areas
   - Command execution
   - Testing: tool integration

5. **Script Console** (Day 9-10)
   - Editor interface
   - Execution controls
   - Output display
   - Testing: script workflow

**Success Criteria:**
- ✅ All views functional and navigable
- ✅ Smooth real-time updates (no flickering)
- ✅ Responsive on terminals 80x24 and larger
- ✅ Intuitive keybindings

### Phase 6: NAT & Advanced Features (Week 10)
**Duration:** 5 days

**Milestones:**
1. **NAT Detection** (Day 1-2)
   - Gateway identification
   - UPnP/NAT-PMP integration
   - Public IP discovery
   - Testing: various NAT scenarios

2. **Multi-subnet Support** (Day 3-4)
   - Route table parsing
   - Cross-subnet discovery
   - Subnet grouping in UI
   - Testing: multi-homed systems

3. **Optimization** (Day 5)
   - Performance profiling
   - Memory optimization
   - Scan strategy tuning
   - Testing: stress tests, benchmarks

**Success Criteria:**
- ✅ Accurate NAT detection
- ✅ Gateway info display
- ✅ Multi-subnet visualization
- ✅ <10s scan maintained

### Phase 7: Testing & Documentation (Week 11-12)
**Duration:** 10 days

**Milestones:**
1. **Comprehensive Testing** (Day 1-5)
   - Unit test coverage >80%
   - Integration tests
   - E2E tests
   - Performance benchmarks
   - Testing: CI/CD validation

2. **Documentation** (Day 6-8)
   - User guide
   - API documentation
   - Script writing guide
   - Architecture docs
   - Testing: doc accuracy

3. **Polish & Bug Fixes** (Day 9-10)
   - Address test failures
   - UI polish
   - Performance fixes
   - Testing: regression tests

**Success Criteria:**
- ✅ All tests passing
- ✅ Complete documentation
- ✅ No critical bugs
- ✅ Ready for release

## 3. Technical Specifications

### 3.1 Performance Requirements

| Metric | Target | Critical Path |
|--------|--------|---------------|
| Network scan time (/24) | <10s | Parallel ARP + ICMP + TCP |
| Device detection rate | 100% | Multi-method scanning |
| Alert latency | <500ms | Event-driven architecture |
| UI refresh rate | 30 FPS | Efficient diff rendering |
| Memory usage (1000 devices) | <100MB | Efficient data structures |
| Startup time | <2s | Lazy initialization |

### 3.2 Network Scanning Specifications

#### 3.2.1 ARP Scan
```go
// Configuration
type ARPScanConfig struct {
    Interface   string        // Network interface to use
    Timeout     time.Duration // Per-request timeout (default: 100ms)
    Retries     int          // Retry count (default: 2)
    Workers     int          // Concurrent workers (default: 256)
}

// Performance
// - 256 concurrent workers
// - 100ms timeout per IP
// - /24 network (256 IPs): ~2-3 seconds
// - Only works on local subnet
// - Most reliable method
```

#### 3.2.2 ICMP Scan
```go
// Configuration
type ICMPScanConfig struct {
    Timeout     time.Duration // Per-ping timeout (default: 1s)
    Count       int          // Pings per host (default: 1)
    Workers     int          // Concurrent workers (default: 256)
    PacketSize  int          // ICMP payload size (default: 32)
}

// Performance
// - 256 concurrent workers
// - 1s timeout per IP
// - /24 network: ~3-4 seconds
// - May be blocked by firewalls
// - Works across subnets
```

#### 3.2.3 TCP Port Scan
```go
// Configuration
type TCPScanConfig struct {
    Ports       []int         // Ports to scan (default: [22,80,443])
    Timeout     time.Duration // Per-connection timeout (default: 500ms)
    Workers     int          // Concurrent workers (default: 512)
}

// Performance
// - 512 concurrent workers
// - 500ms timeout per port
// - 3 ports × 256 IPs: ~2-3 seconds
// - Most reliable for NAT traversal
// - Service detection via banner
```

#### 3.2.4 Passive Discovery
```go
// Configuration
type PassiveDiscoveryConfig struct {
    Interface   string        // Interface to monitor
    Duration    time.Duration // How long to listen (default: 5s)
    Protocols   []string      // Protocols to watch (ARP, mDNS, etc.)
}

// Performance
// - Zero active scanning
// - Discovers chatty devices
// - Background monitoring
// - Complements active scans
```

### 3.3 Data Storage Schema

#### 3.3.1 SQLite Schema
```sql
-- Devices table
CREATE TABLE devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip TEXT NOT NULL,
    mac TEXT,
    hostname TEXT,
    vendor TEXT,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT,
    metadata JSON,
    UNIQUE(ip, mac)
);

CREATE INDEX idx_devices_ip ON devices(ip);
CREATE INDEX idx_devices_last_seen ON devices(last_seen);

-- Device history
CREATE TABLE device_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER,
    event_type TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSON,
    FOREIGN KEY(device_id) REFERENCES devices(id)
);

CREATE INDEX idx_history_device ON device_history(device_id);
CREATE INDEX idx_history_time ON device_history(timestamp);

-- Alerts
CREATE TABLE alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER,
    alert_type TEXT,
    severity INTEGER,
    message TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged BOOLEAN DEFAULT FALSE,
    FOREIGN KEY(device_id) REFERENCES devices(id)
);

CREATE INDEX idx_alerts_device ON alerts(device_id);
CREATE INDEX idx_alerts_ack ON alerts(acknowledged);

-- Tool history
CREATE TABLE tool_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tool_name TEXT,
    arguments TEXT,
    output TEXT,
    exit_code INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Scripts
CREATE TABLE scripts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    content TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    last_executed TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3.4 Configuration File Format

#### 3.4.1 YAML Configuration
```yaml
# ~/.config/tuimap/config.yaml

# Network scanning settings
scanner:
  interface: "" # Auto-detect if empty
  scan_interval: 60s # Background scan frequency
  timeout: 10s
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
    ports: [22, 80, 443, 3389, 5900]

# Alert settings
alerts:
  enabled: true
  rules:
    - type: new_device
      severity: 1
      action: notify
    - type: device_offline
      severity: 2
      action: log
    - type: port_change
      severity: 2
      action: notify

# NAT settings
nat:
  detect: true
  upnp_enabled: true
  public_ip_check: true
  stun_servers:
    - stun.l.google.com:19302
    - stun1.l.google.com:19302

# Scripting settings
scripting:
  enabled: true
  script_dir: ~/.config/tuimap/scripts
  auto_run: []
  max_execution_time: 30s
  max_memory: 50MB

# TUI settings
tui:
  theme: dark
  refresh_rate: 30
  default_view: network_map
  keybindings:
    quit: q
    refresh: r
    scan: s

# Storage settings
storage:
  database: ~/.local/share/tuimap/tuimap.db
  history_retention: 30d
  max_devices: 10000

# Logging
logging:
  level: info # debug, info, warn, error
  file: ~/.local/share/tuimap/tuimap.log
  max_size: 10MB
```

### 3.5 API for Scripting

#### 3.5.1 Network Functions
```go
// Scan functions
scan(subnet string) []Device         // Full network scan
scanARP(subnet string) []Device      // ARP-only scan
scanICMP(subnet string) []Device     // ICMP-only scan
scanTCP(ip string, ports []int) []int // Port scan

// Device queries
getDevices() []Device                // All known devices
getDevice(ip string) Device          // Specific device
findDevices(filter map[string]interface{}) []Device

// Network operations
ping(ip string) bool                 // ICMP ping
resolve(hostname string) string      // DNS lookup
reverseLookup(ip string) string      // Reverse DNS
traceroute(ip string) []Hop          // Traceroute
whois(query string) string           // WHOIS lookup
dig(domain string, recordType string) []string

// Connection testing
tcpConnect(ip string, port int) bool
udpSend(ip string, port int, data []byte) bool
```

#### 3.5.2 Alert Functions
```go
// Alert creation
alert(message string, severity int)
alertDevice(deviceIP string, message string, severity int)

// Alert queries
getAlerts() []Alert
getUnacknowledgedAlerts() []Alert
acknowledgeAlert(id int)
```

#### 3.5.3 Data Functions
```go
// Persistent key-value store
set(key string, value interface{})
get(key string) interface{}
delete(key string)
exists(key string) bool

// Bulk operations
setAll(data map[string]interface{})
getAll() map[string]interface{}
```

#### 3.5.4 Utility Functions
```go
// Time functions
now() time.Time
sleep(duration string)
schedule(interval string, func)

// String operations
format(template string, args ...interface{}) string
match(pattern string, text string) bool

// Logging
log(message string)
debug(message string)
error(message string)
```

### 3.6 Optimization Techniques

#### 3.6.1 Achieving <10s Scan Time

**Strategy Breakdown:**
1. **Parallel Execution (0-9s):**
   - ARP scan: 0-3s (256 workers)
   - ICMP scan: 0-4s (256 workers) - overlapped
   - TCP scan: 0-3s (512 workers) - overlapped
   - All three run simultaneously

2. **Early Exit Optimization:**
   - Cancel remaining scans when 99% confidence achieved
   - Skip IPs that responded to ARP (faster)
   - Adaptive worker pool based on response rate

3. **Smart Subnet Detection:**
   - Pre-calculate subnet mask
   - Skip broadcast and network addresses
   - Prioritize gateway and common IPs first

4. **Response Deduplication (<1s):**
   - Hash-based device identification
   - Merge results from multiple methods
   - Update existing device records

**Code Example:**
```go
func FastNetworkScan(subnet string) ([]Device, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Run all scan methods in parallel
    var wg sync.WaitGroup
    results := make(chan Device, 1000)
    
    // ARP scan (fastest, most reliable)
    wg.Add(1)
    go func() {
        defer wg.Done()
        arpDevices := ARPScan(ctx, subnet, 256)
        for _, d := range arpDevices {
            results <- d
        }
    }()
    
    // ICMP scan (good for cross-subnet)
    wg.Add(1)
    go func() {
        defer wg.Done()
        icmpDevices := ICMPScan(ctx, subnet, 256)
        for _, d := range icmpDevices {
            results <- d
        }
    }()
    
    // TCP scan (works through firewalls)
    wg.Add(1)
    go func() {
        defer wg.Done()
        tcpDevices := TCPScan(ctx, subnet, 512, []int{80, 443, 22})
        for _, d := range tcpDevices {
            results <- d
        }
    }()
    
    // Collect and deduplicate
    go func() {
        wg.Wait()
        close(results)
    }()
    
    deviceMap := make(map[string]Device)
    for device := range results {
        if existing, ok := deviceMap[device.IP.String()]; ok {
            // Merge device info
            deviceMap[device.IP.String()] = mergeDeviceInfo(existing, device)
        } else {
            deviceMap[device.IP.String()] = device
        }
    }
    
    // Convert map to slice
    devices := make([]Device, 0, len(deviceMap))
    for _, d := range deviceMap {
        devices = append(devices, d)
    }
    
    return devices, nil
}
```

#### 3.6.2 Memory Optimization
- Use sync.Pool for packet buffers
- Circular buffer for device history
- Lazy loading of vendor database
- Stream processing for large result sets

#### 3.6.3 CPU Optimization
- GOMAXPROCS tuning
- Worker pool reuse
- Efficient parsing (no regex in hot paths)
- Profile-guided optimization

### 3.7 Security Considerations

1. **Privilege Handling:**
   - Raw socket operations require root/admin
   - Graceful degradation to unprivileged methods
   - Clear error messages for permission issues

2. **Script Sandboxing:**
   - Resource limits (CPU, memory, execution time)
   - No direct file system access
   - No arbitrary command execution
   - Whitelist of allowed operations

3. **Network Safety:**
   - Rate limiting to avoid network DoS
   - Configurable scan aggressiveness
   - Respect robots.txt for WHOIS
   - No exploit attempts

4. **Data Privacy:**
   - Local storage only (no cloud)
   - Optional encryption for database
   - Clear data on request
   - No telemetry

## 4. Progress Tracking

### 4.1 Development Status

#### Overall Progress: 0%

| Phase | Status | Progress | Start Date | End Date | Notes |
|-------|--------|----------|------------|----------|-------|
| Phase 0: Project Setup | 🔄 In Progress | 10% | TBD | TBD | Repository initialized |
| Phase 1: Network Scanner | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 2: Device Tracker | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 3: Network Tools | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 4: Scripting Engine | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 5: TUI Implementation | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 6: NAT & Advanced | ⏳ Pending | 0% | TBD | TBD | - |
| Phase 7: Testing & Docs | ⏳ Pending | 0% | TBD | TBD | - |

**Legend:**
- ⏳ Pending
- 🔄 In Progress
- ✅ Complete
- ❌ Blocked
- ⚠️ At Risk

### 4.2 Feature Completion Checklist

#### Core Features
- [ ] Network scanning (<10s requirement)
  - [ ] ARP scan implementation
  - [ ] ICMP ping sweep
  - [ ] TCP port scanning
  - [ ] Passive discovery
  - [ ] Result aggregation
  - [ ] Performance optimization (target: <10s)
  
- [ ] Device tracking
  - [ ] In-memory device registry
  - [ ] State change detection
  - [ ] Alert engine
  - [ ] Persistent storage
  - [ ] History tracking
  
- [ ] Network tools
  - [ ] netcat implementation
  - [ ] telnet client
  - [ ] traceroute with hop timing
  - [ ] dig (DNS queries)
  - [ ] whois client
  
- [ ] Scripting engine
  - [ ] d5/tengo integration
  - [ ] API bridge
  - [ ] Script management
  - [ ] Example scripts
  - [ ] Documentation
  
- [ ] TUI interface
  - [ ] Network map view
  - [ ] Device list view
  - [ ] Tool execution view
  - [ ] Script console
  - [ ] Keybinding system

#### NAT Environment Features
- [ ] NAT detection
- [ ] Gateway identification
- [ ] UPnP/NAT-PMP integration
- [ ] Public IP discovery
- [ ] Multi-subnet support

#### Quality & Documentation
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests
- [ ] Performance benchmarks
- [ ] User documentation
- [ ] API documentation
- [ ] Example scripts
- [ ] Troubleshooting guide

### 4.3 Performance Metrics Tracking

| Metric | Target | Current | Status | Notes |
|--------|--------|---------|--------|-------|
| Scan Time (/24) | <10s | - | ⏳ | Target is critical requirement |
| Device Detection Rate | 100% | - | ⏳ | Must detect all active devices |
| Alert Latency | <500ms | - | ⏳ | Real-time requirement |
| Memory Usage (1k devices) | <100MB | - | ⏳ | Efficiency target |
| UI Refresh Rate | 30 FPS | - | ⏳ | Smooth experience |
| Startup Time | <2s | - | ⏳ | Quick launch |

### 4.4 Risk Register

| Risk | Probability | Impact | Mitigation | Status |
|------|-------------|--------|------------|--------|
| Raw socket permissions | High | High | Provide unprivileged fallback methods | Open |
| Scan time exceeds 10s on large networks | Medium | High | Implement adaptive scanning, early exit | Open |
| NAT traversal complexity | Medium | Medium | Focus on local subnet, document limitations | Open |
| Tengo script security | Low | High | Implement strict sandboxing, resource limits | Open |
| Cross-platform compatibility | Medium | Medium | Test on Linux, macOS, Windows; use build tags | Open |
| Memory leaks in long-running scans | Low | Medium | Implement proper cleanup, use profiling | Open |

### 4.5 Testing Strategy

#### Unit Testing
- All core functions have tests
- Mock network interfaces for testing
- Coverage target: >80%
- Run on every commit

#### Integration Testing
- End-to-end scan workflows
- Tool integration tests
- Script execution tests
- Run before merges

#### Performance Testing
- Scan time benchmarks on various network sizes
- Memory profiling with large device counts
- Stress testing with rapid device changes
- Run weekly on dedicated test environment

#### Manual Testing
- UI/UX validation
- Cross-platform verification
- Real network testing
- NAT environment testing

### 4.6 Release Milestones

#### v0.1.0 - Alpha (End of Phase 3)
**Target Date:** Week 6
**Features:**
- Basic network scanning
- Device tracking
- Network tools
- CLI interface (no TUI yet)

#### v0.2.0 - Beta (End of Phase 5)
**Target Date:** Week 9
**Features:**
- Full TUI interface
- All network tools
- Basic scripting
- Alert system

#### v0.3.0 - RC (End of Phase 6)
**Target Date:** Week 10
**Features:**
- NAT optimization
- Multi-subnet support
- Performance tuning
- Advanced scripting

#### v1.0.0 - Release (End of Phase 7)
**Target Date:** Week 12
**Features:**
- Complete feature set
- Full documentation
- Comprehensive tests
- Production-ready

### 4.7 Success Criteria

**Must Have (for v1.0):**
1. ✅ Scan time: <10 seconds for /24 network
2. ✅ Device detection: 100% of active devices
3. ✅ All 5 network tools functional (netcat, telnet, traceroute, dig, whois)
4. ✅ d5/tengo scripting fully integrated
5. ✅ Real-time network map view
6. ✅ Device tracking with alerts
7. ✅ Works in NAT environments
8. ✅ Cross-platform support (Linux, macOS, Windows)

**Should Have:**
- Export capabilities (JSON, CSV)
- Configurable scan strategies
- Script library
- Dark/light theme support
- Network traffic visualization

**Nice to Have:**
- Cloud integration (for multi-site monitoring)
- Mobile companion app
- Plugin system
- Advanced analytics
- ML-based anomaly detection

### 4.8 Maintenance Plan

**Post-Release:**
- Bug fix releases: as needed
- Feature releases: quarterly
- Security updates: immediate
- Dependency updates: monthly review

**Community:**
- GitHub issues for bug reports
- Discussions for feature requests
- Wiki for community scripts
- Contributing guidelines

---

## Appendix

### A. Useful Commands

```bash
# Build
go build -o tuimap cmd/tuimap/main.go

# Run with elevated privileges (for raw sockets)
sudo ./tuimap

# Run with specific interface
./tuimap --interface eth0

# Run in debug mode
./tuimap --debug

# Export device list
./tuimap export --format json > devices.json

# Run a script
./tuimap script run my_script.tengo

# Generate config file
./tuimap config init
```

### B. Architecture Decision Records

#### ADR-001: Choose Bubble Tea for TUI
**Decision:** Use charmbracelet/bubbletea for TUI framework
**Rationale:** 
- Modern Go-based framework
- Excellent documentation
- Active community
- Elegant design patterns
**Alternatives Considered:** tview, termui
**Status:** Accepted

#### ADR-002: Choose Tengo for Scripting
**Decision:** Use d5/tengo for embedded scripting
**Rationale:**
- Go-based (easy integration)
- Secure sandbox
- Good performance
- Simple syntax (like Go)
**Alternatives Considered:** Lua (gopher-lua), JavaScript (goja)
**Status:** Accepted

#### ADR-003: Multi-Method Scanning
**Decision:** Use parallel ARP, ICMP, and TCP scanning
**Rationale:**
- Achieves <10s scan time
- 100% detection rate
- Works in various network configurations
- Handles firewall scenarios
**Alternatives Considered:** Single-method scanning, sequential scanning
**Status:** Accepted

#### ADR-004: SQLite for Persistence
**Decision:** Use SQLite for device history and alerts
**Rationale:**
- Embedded (no separate database server)
- Reliable and well-tested
- Good performance for our use case
- Easy backup (single file)
**Alternatives Considered:** BoltDB, in-memory only
**Status:** Accepted

### C. References

**Network Scanning:**
- [Nmap Documentation](https://nmap.org/book/)
- [TCP/IP Illustrated](https://www.amazon.com/TCP-Illustrated-Vol-Addison-Wesley-Professional/dp/0201633469)
- [ARP Protocol (RFC 826)](https://tools.ietf.org/html/rfc826)
- [ICMP Protocol (RFC 792)](https://tools.ietf.org/html/rfc792)

**Go Libraries:**
- [gopacket Documentation](https://pkg.go.dev/github.com/google/gopacket)
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Tengo Documentation](https://github.com/d5/tengo/blob/master/docs/tutorial.md)

**Network Tools:**
- [Netcat Guide](http://nc110.sourceforge.net/)
- [Dig Manual](https://linux.die.net/man/1/dig)
- [Traceroute Implementation](https://github.com/aeden/traceroute)

**NAT Traversal:**
- [UPnP IGD Protocol](https://en.wikipedia.org/wiki/Internet_Gateway_Device_Protocol)
- [NAT-PMP Protocol (RFC 6886)](https://tools.ietf.org/html/rfc6886)
- [STUN Protocol (RFC 5389)](https://tools.ietf.org/html/rfc5389)

---

*Last Updated: 2025-10-21*
*Version: 1.0*
*Status: Planning Phase*
