# Getting Started with TuiMap

Welcome to TuiMap! This guide will help you get started with the terminal-based network diagnostic and mapping tool.

## Prerequisites

- Go 1.25 or higher (for building from source)
- Root/admin privileges (for full scanning capabilities)
- Linux, macOS, or Windows

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/opd-ai/tuimap.git
cd tuimap

# Build the binary
make build

# The binary will be at ./bin/tuimap
```

### Install to System

```bash
# Install to GOPATH/bin
make install

# Or manually copy
sudo cp bin/tuimap /usr/local/bin/
```

## First Run

### 1. Initialize Configuration

Create the default configuration file:

```bash
tuimap config init
```

This creates `~/.config/tuimap/config.yaml` with sensible defaults.

### 2. View Configuration

```bash
tuimap config show
```

### 3. Run TuiMap

```bash
# Run with default settings (requires root for full capabilities)
sudo tuimap

# Run with specific interface
sudo tuimap --interface eth0

# Run in debug mode
sudo tuimap --debug
```

## Configuration

The configuration file is located at `~/.config/tuimap/config.yaml`.

### Scanner Settings

```yaml
scanner:
  interface: ""           # Auto-detect if empty
  scan_interval: 60s      # Background scan frequency
  timeout: 10s            # Total scan timeout
  methods:                # Scanning methods to use
    - arp                 # Fastest, local subnet only (requires root)
    - icmp                # Cross-subnet capability (requires root)
    - tcp                 # Works without root
  arp:
    workers: 256          # Concurrent workers for ARP
    timeout: 100ms        # Per-host timeout
    retries: 2            # Retry count
  icmp:
    workers: 256          # Concurrent workers for ICMP
    timeout: 1s           # Per-host timeout
    count: 1              # Pings per host
  tcp:
    workers: 512          # Concurrent workers for TCP
    timeout: 500ms        # Per-connection timeout
    ports: [22, 80, 443, 3389, 5900]  # Ports to scan
```

### Alert Rules

```yaml
alerts:
  enabled: true
  rules:
    - type: new_device     # Alert on new devices
      severity: 2
      action: notify
    - type: device_offline # Alert when devices go offline
      severity: 1
      action: log
    - type: port_change    # Alert on port changes
      severity: 2
      action: notify
    - type: mac_conflict   # Alert on MAC conflicts
      severity: 3
      action: notify
```

### NAT Settings

```yaml
nat:
  detect: true            # Detect NAT environment
  upnp_enabled: true      # Use UPnP for gateway info
  public_ip_check: true   # Check public IP via STUN
  stun_servers:
    - stun.l.google.com:19302
    - stun1.l.google.com:19302
```

### TUI Settings

```yaml
tui:
  refresh_rate: 30        # FPS for UI updates
  theme: dark             # Color theme
  keybindings:
    quit: q
    scan: s
    refresh: r
```

### Scripting Settings

```yaml
scripting:
  enabled: true
  max_execution_time: 30s # Maximum script runtime
  max_memory: "50MB"      # 50MB memory limit
  script_dir: ~/.config/tuimap/scripts
```

## TUI Navigation

Once TuiMap is running, use these keyboard shortcuts:

| Key | Action |
|-----|--------|
| `1` | Network Map View |
| `2` | Device List View |
| `3` | Network Tools View |
| `4` | Script Console View |
| `s` | Start Network Scan |
| `n` | Next Subnet |
| `r` | Refresh Display |
| `q` | Quit |

## Network Tools

TuiMap includes integrated network diagnostic tools:

### Netcat
TCP/UDP connection testing:
```
nc <host> <port> [--udp] [--data <text>]
```

### Telnet
Protocol testing and banner grabbing:
```
telnet <host> [port]
```

### Traceroute
Path discovery with hop timing:
```
traceroute <host> [--max-hops <n>]
```

### Dig
DNS query interface:
```
dig <domain> [type] [@server]
```
Types: A, AAAA, MX, TXT, NS, CNAME

### Whois
Domain/IP registration lookup:
```
whois <domain|ip> [--server <server>]
```

## Scripting

TuiMap supports automation via Tengo scripts. Scripts are located in `~/.config/tuimap/scripts/`.

### Available Script APIs

**Network Operations:**
- `scan()` - Run network scan
- `ping(host)` - Ping a host
- `portScan(host, ports)` - Scan specific ports
- `resolve(hostname)` - DNS resolution

**Device Management:**
- `getDevices()` - Get all tracked devices
- `alert(level, message)` - Generate alert

**Storage:**
- `set(key, value)` - Store persistent value
- `get(key)` - Retrieve value

### Example Script

```tengo
// alert_on_new_ssh.tengo
// Alert when a new device with SSH port is found

devices := getDevices()
for device in devices {
    if 22 in device.ports && device.status == "new" {
        alert("new_device", "New SSH server: " + device.ip)
    }
}
```

## Permissions

TuiMap requires different permission levels for different features:

| Feature | Permission | Reason |
|---------|------------|--------|
| ARP Scanning | Root/Admin | Raw socket access |
| ICMP Scanning | Root/Admin | Raw socket access |
| TCP Scanning | User | Standard TCP connections |
| Network Tools | Varies | Some need elevated privileges |
| TUI Interface | User | No special permissions |

When run without root, TuiMap gracefully falls back to unprivileged methods (TCP scanning only).

## Troubleshooting

### "Permission denied" errors
Run with `sudo` or grant `CAP_NET_RAW` capability:
```bash
sudo setcap cap_net_raw+ep /usr/local/bin/tuimap
```

### Scans taking too long
- Reduce worker counts in configuration
- Use TCP-only scanning (fastest without root)
- Scan smaller subnets

### No devices found
- Verify network interface is correct
- Check firewall settings
- Try running with `--debug` flag

### Configuration issues
- Run `tuimap config show` to verify settings
- Reset by removing the config file and running `tuimap config init`

## License

TuiMap is licensed under the MIT License. See [LICENSE](../LICENSE) for details.
