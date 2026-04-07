# TuiMap User Guide

This comprehensive guide covers all features and usage patterns for TuiMap, the terminal-based network diagnostic and mapping tool.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Network Scanning](#network-scanning)
4. [Device Tracking](#device-tracking)
5. [Alert System](#alert-system)
6. [Network Tools](#network-tools)
7. [Scripting](#scripting)
8. [TUI Interface](#tui-interface)
9. [Advanced Configuration](#advanced-configuration)
10. [Best Practices](#best-practices)

## Overview

TuiMap is designed for fast, accurate network discovery and monitoring. Key features include:

- **<10 Second Scanning**: Discover all devices on /24 networks rapidly
- **Multi-Method Discovery**: ARP, ICMP, and TCP scanning in parallel
- **Real-Time Tracking**: Monitor device status changes continuously
- **Integrated Tools**: Built-in netcat, telnet, traceroute, dig, whois
- **Scripting**: Automate tasks with Tengo scripts
- **NAT Support**: Optimized for NAT environments

## Installation

### From Source

```bash
git clone https://github.com/opd-ai/tuimap.git
cd tuimap
make build
sudo make install
```

### Binary Only

```bash
./bin/tuimap --help
```

### Verify Installation

```bash
tuimap version
```

## Network Scanning

### Scanning Methods

TuiMap uses three complementary scanning methods:

#### ARP Scanning (Layer 2)
- Fastest method
- Works only on local subnet
- Discovers MAC addresses
- Requires root privileges

#### ICMP Scanning (Layer 3)
- Cross-subnet capability
- Standard ping sweep
- Requires root privileges

#### TCP Scanning (Layer 4)
- Works through firewalls
- Detects open ports
- No root required

### Starting a Scan

```bash
# Auto-detect subnet
sudo tuimap scan

# Specific subnet
sudo tuimap scan --subnet 192.168.1.0/24

# Specific interface
sudo tuimap scan --interface eth0

# TCP-only (no root needed)
tuimap scan --methods tcp
```

### Scan Performance

Target: Complete /24 network scan in <10 seconds

| Method | Typical Time | Workers | Timeout |
|--------|-------------|---------|---------|
| ARP | 0-3s | 256 | 100ms |
| ICMP | 0-4s | 256 | 1s |
| TCP | 0-3s | 512 | 500ms |

### Multi-Subnet Scanning

TuiMap can discover and scan multiple subnets:

```bash
# Scan all discovered subnets
sudo tuimap scan --all-subnets

# Scan from routing table
sudo tuimap scan --from-routes
```

## Device Tracking

### Device States

| State | Description |
|-------|-------------|
| `new` | Just discovered |
| `online` | Currently responding |
| `offline` | Not seen recently |
| `changed` | Ports or attributes changed |

### Tracking Configuration

```yaml
tracker:
  offline_threshold: 5m    # Time before marking offline
  history_retention: 168h  # 7 days of history
```

### Viewing Devices

In the TUI, press `2` for Device List View:
- Sort by IP, MAC, hostname, last seen
- Filter by status, vendor, ports
- Select device for details

## Alert System

### Alert Types

| Type | Severity | Trigger |
|------|----------|---------|
| `new_device` | 2 | Unknown device appears |
| `device_offline` | 1 | Device stops responding |
| `port_change` | 2 | Open ports changed |
| `mac_conflict` | 3 | Same IP, different MAC |

### Configuring Alerts

```yaml
alerts:
  enabled: true
  rules:
    - type: new_device
      severity: 2
      action: notify
      message: "New device detected: ${ip}"
    
    - type: device_offline
      severity: 1
      action: log
      
    - type: port_change
      severity: 2
      action: notify
      
    - type: mac_conflict
      severity: 3
      action: notify
      message: "MAC conflict detected for ${ip}"
```

### Alert Actions

- `notify`: Show in TUI alert panel
- `log`: Write to log file
- `script`: Run custom script (future)

## Network Tools

### Netcat

TCP/UDP connection testing and data transfer.

```
# Basic TCP connection
nc localhost 80

# UDP mode
nc --udp localhost 53

# Send data
nc localhost 80 --data "GET / HTTP/1.0\r\n\r\n"
```

### Telnet

Protocol testing with banner grabbing.

```
# Default port 23
telnet router.local

# Custom port
telnet webserver.local 80
```

### Traceroute

Path discovery showing each hop.

```
# Standard traceroute
traceroute google.com

# Limit hops
traceroute google.com --max-hops 15
```

### Dig

DNS query tool supporting multiple record types.

```
# A record (default)
dig example.com

# MX records
dig example.com MX

# Use specific DNS server
dig example.com @8.8.8.8

# TXT records
dig example.com TXT
```

### Whois

Domain and IP registration lookup.

```
# Domain lookup
whois example.com

# IP lookup
whois 8.8.8.8

# Custom server
whois example.com --server whois.example.org
```

## Scripting

### Script Location

Scripts are stored in `~/.config/tuimap/scripts/`.

### Tengo Language

TuiMap uses [Tengo](https://github.com/d5/tengo), a fast, Go-like scripting language.

### Available APIs

#### Network Functions

```tengo
// Run a network scan
result := scan()
fmt.println("Found", len(result.devices), "devices")

// Ping a host
rtt := ping("192.168.1.1")
if rtt > 0 {
    fmt.println("Latency:", rtt, "ms")
}

// Scan specific ports
ports := port_scan("192.168.1.1", [22, 80, 443])
for port in ports {
    fmt.println("Port", port, "is open")
}

// DNS resolution
ips := resolve("example.com")
```

#### Device Management

```tengo
// Get all devices
devices := get_devices()
for device in devices {
    fmt.println(device.ip, device.hostname, device.status)
}

// Get specific device
device := get_device("192.168.1.1")
if device {
    fmt.println("Hostname:", device.hostname)
    fmt.println("Vendor:", device.vendor)
    fmt.println("Ports:", device.ports)
}

// Generate alert
alert("new_device", "Custom alert message")
```

#### Persistent Storage

```tengo
// Store a value
set("last_scan_count", 42)

// Retrieve value
count := get("last_scan_count")

// Delete value
delete("last_scan_count")
```

### Example Scripts

#### Monitor Specific Ports

```tengo
// monitor_ports.tengo
// Alert if important ports are open on unexpected devices

important_ports := [22, 3389, 5900]  // SSH, RDP, VNC
allowed_ips := ["192.168.1.1", "192.168.1.10"]

devices := get_devices()
for device in devices {
    if device.ip in allowed_ips {
        continue
    }
    
    for port in important_ports {
        if port in device.ports {
            alert("port_change", 
                  "Unexpected " + string(port) + " on " + device.ip)
        }
    }
}
```

#### Track Device Counts

```tengo
// device_stats.tengo
// Track device counts over time

devices := get_devices()
online_count := 0
for device in devices {
    if device.status == "online" {
        online_count++
    }
}

// Store for trending
timestamp := time.now()
key := "device_count_" + string(timestamp)
set(key, online_count)

fmt.println("Online devices:", online_count)
```

### Script Limits

| Limit | Value | Purpose |
|-------|-------|---------|
| Execution time | 30s | Prevent runaway scripts |
| Memory | 50MB | Prevent memory exhaustion |
| File access | None | Sandboxed execution |
| System commands | None | Security isolation |

## TUI Interface

### Views

| Key | View | Description |
|-----|------|-------------|
| `1` | Network Map | Visual network topology |
| `2` | Device List | Sortable device table |
| `3` | Tools | Network tool interface |
| `4` | Scripts | Script console |

### Network Map View

Shows network topology with:
- Gateway at center
- Devices arranged by subnet
- Status indicators (green=online, red=offline, yellow=new)
- Connection lines

### Device List View

Sortable columns:
- IP Address
- MAC Address
- Hostname
- Vendor
- Status
- Ports
- Last Seen

Press `f` to filter by any column.

### Tools View

Tab interface for each tool:
- Input area for commands
- Scrollable output
- Command history (up/down arrows)

### Script Console

- Script file browser
- Editor area
- Execution output
- Run/Stop controls

### Global Keys

| Key | Action |
|-----|--------|
| `q` | Quit |
| `s` | Start scan |
| `r` | Refresh |
| `f` | Filter |
| `?` | Help |
| `Tab` | Next pane |
| `Shift+Tab` | Previous pane |

## Advanced Configuration

### Performance Tuning

For large networks (>1000 devices):

```yaml
scanner:
  timeout: 15s            # Allow more time
  arp:
    workers: 512          # More parallel workers
  icmp:
    workers: 512
  tcp:
    workers: 1024
    
tracker:
  offline_threshold: 10m  # Longer threshold
  history_retention: 24h  # Shorter retention
  
storage:
  memory_limit: 200MB     # More memory for devices
```

### Low-Bandwidth Networks

```yaml
scanner:
  timeout: 20s
  arp:
    workers: 64           # Fewer workers
    timeout: 500ms        # Longer timeouts
  icmp:
    workers: 64
    timeout: 2s
  tcp:
    workers: 128
    timeout: 2s
```

### Stealth Mode

For security scanning where detection should be minimized:

```yaml
scanner:
  methods:
    - tcp                 # TCP only, no broadcast
  tcp:
    workers: 32           # Slow and steady
    timeout: 1s
    ports: [80, 443]      # Common ports only
```

### Logging

```yaml
logging:
  level: info             # debug, info, warn, error
  file: ~/.local/share/tuimap/tuimap.log
  max_size: 10MB
  max_backups: 5
```

## Best Practices

### Security

1. **Minimal Privileges**: Run without root when TCP scanning is sufficient
2. **Review Scripts**: Only run scripts you understand and trust
3. **Alert on Changes**: Enable `new_device` and `mac_conflict` alerts
4. **Limit Scan Scope**: Scan only networks you're authorized to scan

### Performance

1. **Start TCP-Only**: Test without root first
2. **Profile Your Network**: Adjust timeouts based on actual latency
3. **Use Multi-Subnet Carefully**: Each subnet adds scan time
4. **Monitor Memory**: Large networks need more resources

### Reliability

1. **Check Logs**: Review logs for scan errors
2. **Verify Detection**: Cross-reference with other tools initially
3. **Test Alerts**: Ensure alert rules trigger correctly
4. **Regular Updates**: Keep TuiMap and dependencies updated

### Monitoring

1. **Consistent Intervals**: Use regular scan intervals for trending
2. **Baseline First**: Establish normal device counts
3. **Document Known Devices**: Use scripts to track expected devices
4. **Export History**: Periodically export device history

## Appendix: Configuration Reference

Full configuration file with all options:

```yaml
# Scanner configuration
scanner:
  interface: ""           # Network interface (auto-detect if empty)
  scan_interval: 60s      # Background scan interval
  timeout: 10s            # Total scan timeout budget
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

# Alert configuration
alerts:
  enabled: true
  rules:
    - type: new_device
      severity: 2
      action: notify
    - type: device_offline
      severity: 1
      action: log
    - type: port_change
      severity: 2
      action: notify
    - type: mac_conflict
      severity: 3
      action: notify

# NAT configuration
nat:
  detect: true
  upnp_enabled: true
  nat_pmp_enabled: true
  public_ip_check: true
  stun_servers:
    - stun.l.google.com:19302

# TUI configuration
tui:
  refresh_rate: 30
  theme: default
  keybindings:
    quit: q
    scan: s
    refresh: r
    filter: f
    help: "?"
    view_map: "1"
    view_list: "2"
    view_tools: "3"
    view_scripts: "4"

# Scripting configuration
scripting:
  enabled: true
  max_execution_time: 30s
  max_memory: 52428800
  scripts_dir: ~/.config/tuimap/scripts

# Storage configuration
storage:
  database: ~/.local/share/tuimap/tuimap.db
  history_retention: 168h

# Logging configuration
logging:
  level: info
  file: ""
```
