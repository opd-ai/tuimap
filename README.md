# TuiMap

[![CI](https://img.shields.io/github/actions/workflow/status/opd-ai/tuimap/ci.yml?branch=main&label=CI)](https://github.com/opd-ai/tuimap/actions/workflows/ci.yml) [![Go Version](https://img.shields.io/github/go-mod-go-version/opd-ai/tuimap)](go.mod) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

TuiMap is a terminal-based network diagnostic and mapping tool built in Go.
It discovers devices on local networks using parallel ARP, ICMP, and TCP
scanning, provides real-time device tracking with alerts, integrates
network diagnostic tools, and supports user automation through an embedded
Tengo scripting engine — all within an interactive terminal UI.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Scripting](#scripting)
- [Project Structure](#project-structure)
- [Development](#development)
- [Known Limitations](#known-limitations)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Parallel Multi-Method Scanning** — Runs ARP (layer 2, Linux only),
  ICMP (layer 3), and TCP connect scans simultaneously with configurable
  worker pools and deduplicates results by MAC or IP
  (`internal/scanner/orchestrator.go`)
- **Multi-Subnet Discovery** — Automatically discovers local subnets from
  network interfaces or the system routing table and scans them in parallel
  (`internal/scanner/multisubnet.go`)
- **Device Tracking and Alerts** — Maintains a thread-safe in-memory
  registry of discovered devices, detects status changes (new, online,
  offline, changed), and triggers alerts for new devices, offline devices,
  port changes, and MAC conflicts (`internal/tracker/registry.go`)
- **Persistent Storage** — Stores device history and alerts in a bbolt
  database with configurable retention (`internal/tracker/storage.go`)
- **Integrated Network Tools** — Built-in implementations of netcat
  (`internal/tools/netcat.go`), telnet (`internal/tools/telnet.go`),
  traceroute (`internal/tools/traceroute.go`), dig
  (`internal/tools/dig.go`), and whois (`internal/tools/whois.go`)
- **Tengo Scripting Engine** — Embedded scripting with sandboxed execution,
  configurable time and memory limits, and API access to scanning, pinging,
  port scanning, DNS resolution, alerts, device data, and key-value storage
  (`internal/script/engine.go`)
- **Interactive TUI** — Four-view Bubble Tea interface: Network Map
  (topology visualization), Device List (sortable table), Tool View
  (run diagnostic tools), and Script Console (execute Tengo scripts)
  (`internal/tui/app.go`)
- **NAT Environment Support** — Detects NAT type via STUN (RFC 5389),
  discovers UPnP IGD gateways and NAT-PMP support, and reports external
  IP (`internal/nat/nat.go`)

## Requirements

- **Go** 1.25.0 or higher
- **Linux** for ARP scanning (uses `gopacket/afpacket`; other platforms
  fall back to ICMP and TCP scanning)
- **Root/admin privileges** for ARP and ICMP raw socket operations
  (the tool degrades to TCP-only scanning without elevated permissions)
- **Make** (optional, for Makefile targets)

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/opd-ai/tuimap.git
   cd tuimap
   ```

2. Build the binary (output to `./bin/tuimap`):

   ```bash
   make build
   ```

3. (Optional) Install to `$GOPATH/bin`:

   ```bash
   make install
   ```

The build uses `CGO_ENABLED=0` for a fully static binary. Version, commit
hash, and build date are embedded via linker flags automatically.

## Usage

### Initialize Configuration

Create a default configuration file at `~/.config/tuimap/config.yaml`:

```bash
tuimap config init
```

### Launch the TUI

Start the interactive terminal interface with auto-detected subnet:

```bash
sudo tuimap
```

Specify a network interface:

```bash
sudo tuimap --interface eth0
```

Enable debug logging:

```bash
sudo tuimap --debug
```

### Headless Scan

Scan the auto-detected subnet and print results to stdout:

```bash
# Text output (default)
sudo tuimap scan

# JSON output
sudo tuimap scan --output json

# Scan a specific subnet with a custom timeout
sudo tuimap scan --subnet 192.168.1.0/24 --timeout 20

# Discover and scan all local subnets
sudo tuimap scan --all-subnets

# Scan subnets from the system routing table
sudo tuimap scan --from-routes
```

### View Version

```bash
tuimap version
```

### CLI Reference

| Command | Description |
|---------|-------------|
| `tuimap` | Launch the interactive TUI |
| `tuimap version` | Print version, commit, and build date |
| `tuimap config init` | Create default configuration file |
| `tuimap config show` | Display current configuration as YAML |
| `tuimap scan` | Scan network in headless mode |

**Global flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `~/.config/tuimap/config.yaml` | Path to configuration file |
| `--debug` | `-d` | `false` | Enable debug mode |
| `--interface` | `-i` | auto-detect | Network interface to use |
| `--no-tui` | | `false` | Run root command in headless mode |

**Scan flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--subnet` | `-s` | auto-detect | Subnet in CIDR notation (e.g., `192.168.1.0/24`) |
| `--output` | `-o` | `text` | Output format: `text` or `json` |
| `--timeout` | `-t` | `15` | Scan timeout in seconds |
| `--all-subnets` | | `false` | Discover and scan all local subnets |
| `--from-routes` | | `false` | Scan subnets from the system routing table |

## Configuration

Configuration file: `~/.config/tuimap/config.yaml`

Database file: `~/.local/share/tuimap/tuimap.db`

Log file: `~/.local/share/tuimap/tuimap.log`

Run `tuimap config init` to generate the default file, then edit as needed.
The configuration is managed via Viper and supports YAML format
(`internal/config/config.go`).

### Scanner Settings

| Key | Default | Description |
|-----|---------|-------------|
| `scanner.interface` | `""` (auto-detect) | Network interface name |
| `scanner.scan_interval` | `60s` | Interval between automatic scans |
| `scanner.timeout` | `10s` | Scan timeout |
| `scanner.methods` | `["arp", "icmp", "tcp"]` | Enabled scan methods |
| `scanner.arp.workers` | `256` | ARP scan worker count |
| `scanner.arp.timeout` | `100ms` | Per-host ARP timeout |
| `scanner.arp.retries` | `2` | ARP retry count |
| `scanner.icmp.workers` | `256` | ICMP scan worker count |
| `scanner.icmp.timeout` | `1s` | Per-host ICMP timeout |
| `scanner.icmp.count` | `1` | ICMP echo count per host |
| `scanner.tcp.workers` | `512` | TCP scan worker count |
| `scanner.tcp.timeout` | `500ms` | Per-host TCP connect timeout |
| `scanner.tcp.ports` | `[22, 80, 443, 3389, 5900]` | TCP ports to scan |

### Alert Settings

| Key | Default | Description |
|-----|---------|-------------|
| `alerts.enabled` | `true` | Enable alert system |
| `alerts.rules` | See below | List of alert rules |

Default alert rules:

| Type | Severity | Action |
|------|----------|--------|
| `new_device` | `1` | `notify` |
| `device_offline` | `2` | `log` |
| `port_change` | `2` | `notify` |

### NAT Settings

| Key | Default | Description |
|-----|---------|-------------|
| `nat.detect` | `true` | Enable NAT detection |
| `nat.upnp_enabled` | `true` | Enable UPnP discovery |
| `nat.public_ip_check` | `true` | Check public IP via STUN |
| `nat.stun_servers` | `["stun.l.google.com:19302", "stun1.l.google.com:19302"]` | STUN servers for external IP discovery |

### Scripting Settings

| Key | Default | Description |
|-----|---------|-------------|
| `scripting.enabled` | `true` | Enable scripting engine |
| `scripting.script_dir` | `~/.config/tuimap/scripts` | Directory for user scripts |
| `scripting.auto_run` | `[]` | Scripts to run automatically |
| `scripting.max_execution_time` | `30s` | Maximum script execution time |
| `scripting.max_memory` | `50MB` | Maximum script memory allocation |

### TUI Settings

| Key | Default | Description |
|-----|---------|-------------|
| `tui.theme` | `dark` | Color theme |
| `tui.refresh_rate` | `30` | UI refresh rate in FPS |
| `tui.default_view` | `network_map` | Initial view on startup |
| `tui.keybindings` | `{quit: q, refresh: r, scan: s}` | Custom keybindings |

### Storage Settings

| Key | Default | Description |
|-----|---------|-------------|
| `storage.database` | `~/.local/share/tuimap/tuimap.db` | bbolt database path |
| `storage.history_retention` | `720h` (30 days) | Device history retention period |
| `storage.max_devices` | `10000` | Maximum tracked devices |

### Logging Settings

| Key | Default | Description |
|-----|---------|-------------|
| `logging.level` | `info` | Log level: debug, info, warn, error |
| `logging.file` | `~/.local/share/tuimap/tuimap.log` | Log file path |
| `logging.max_size` | `10MB` | Maximum log file size |

## Scripting

TuiMap embeds a [Tengo](https://github.com/d5/tengo) scripting engine for
user-defined automation. Scripts run in a sandboxed environment with
configurable time (default 30s) and memory (default 50MB) limits. Scripts
have no direct file system or command execution access
(`internal/script/engine.go`).

### Script API Functions

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `scan(subnet)` | `string` | `[]map` | Scan a subnet, returns device maps |
| `ping(host)` | `string` | `map{ok, rtt}` | Ping a host, returns ok (bool) and rtt in ms |
| `portScan(host, ports)` | `string, []int` | `[]int` | Scan TCP ports, returns open ports |
| `resolve(hostname)` | `string` | `[]string` | DNS lookup, returns IP addresses |
| `alert(type, message)` | `string, string` | — | Create an alert |
| `getDevices()` | — | `[]map` | Get all tracked devices |
| `get(key)` | `string` | `any` | Retrieve value from persistent storage |
| `set(key, value)` | `string, any` | — | Store value in persistent storage |
| `print(...)` | `any` | — | Print without newline |
| `println(...)` | `any` | — | Print with newline |

### Example Scripts

Five example scripts are provided in `scripts/examples/`:

- **auto-scan.tengo** — Automatic scanning with alerts for new devices and
  suspicious ports
- **port-monitor.tengo** — Monitors expected ports per device and alerts on
  changes
- **device-inventory.tengo** — Lists tracked devices and counts by status
- **new-device-watcher.tengo** — Detects and logs newly discovered devices
- **health-check.tengo** — Runs diagnostics and generates a health score

Place custom scripts in `~/.config/tuimap/scripts/` and run them from the
TUI Script Console (press `4`).

## Project Structure

```
tuimap/
├── cmd/tuimap/          # CLI entry point and command definitions
├── internal/
│   ├── config/          # Viper-based YAML configuration management
│   ├── nat/             # NAT detection via STUN, UPnP, NAT-PMP
│   ├── scanner/         # ARP, ICMP, TCP scanners and orchestrator
│   ├── script/          # Tengo scripting engine and API bridge
│   ├── tools/           # Network tools: netcat, telnet, traceroute, dig, whois
│   ├── tracker/         # Device registry, alert engine, bbolt storage
│   └── tui/             # Bubble Tea views and UI components
├── pkg/api/             # Public API interfaces and types
├── scripts/examples/    # Example Tengo scripts
├── docs/                # User guide, API reference, troubleshooting
├── Makefile             # Build, test, lint, and install targets
├── go.mod               # Go module (requires Go 1.25.0)
├── ROADMAP.md           # Implementation roadmap and progress
└── GAPS.md              # Known implementation gaps
```

## Development

### Prerequisites

- Go 1.25.0 or higher
- Make (optional)
- golangci-lint (optional, for linting)

### Build and Test

```bash
# Build the binary to ./bin/tuimap
make build

# Run tests with coverage
make test

# Run tests with race detector (requires CGO_ENABLED=1)
make test-race

# Generate HTML coverage report
make test-coverage

# Format code
make fmt

# Run go vet
make vet

# Run golangci-lint (if installed)
make lint

# Download dependencies
make deps

# Clean build artifacts
make clean
```

### CI/CD

The repository includes two GitHub Actions workflows:

- **CI** (`.github/workflows/ci.yml`) — Runs on push and pull requests to
  `main`/`master`. Executes `go build`, `go vet`, `go test -race`, and
  golangci-lint. Checks test coverage against a 35% threshold.
- **Performance Benchmarks** (`.github/workflows/benchmark.yml`) — Runs
  weekly and on pull requests that modify `internal/scanner/` or
  `internal/tracker/`. Validates scan time stays within a 10-second
  threshold.

## Known Limitations

- **ARP Scanning** — ARP scanning uses `gopacket/afpacket` and is only
  available on Linux. On other platforms, `NewARPScanner` returns an error
  and the orchestrator falls back to ICMP and TCP scanning.
- **NAT Port Mapping** — NAT detection (type classification, external IP
  via STUN, UPnP/NAT-PMP discovery) is functional. However,
  `AddPortMapping()` returns `ErrNATUnsupported` because UPnP IGD SOAP
  and NAT-PMP mapping requests are stub implementations.
  `RemovePortMapping()` deletes the local mapping record but performs
  no actual gateway operation.
- **Root/Admin Privileges** — ARP and ICMP scanning require raw socket
  access (`CAP_NET_RAW` on Linux, admin on other platforms). Without
  elevated permissions, the tool degrades to TCP connect scanning only.

## Contributing

Contributions are welcome. To get started:

1. Fork the repository and create a feature branch
2. Run `make test` and `make vet` to verify your changes
3. Submit a pull request against `main`

See [ROADMAP.md](ROADMAP.md) for planned work and [GAPS.md](GAPS.md) for
known implementation gaps.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for
details.

Donate Monero(The only good cryptocurrency) to support development
==================================================================

 - `monero:43H3Uqnc9rfEsJjUXZYmam45MbtWmREFSANAWY5hijY4aht8cqYaT2BCNhfBhua5XwNdx9Tb6BEdt4tjUHJDwNW5H7mTiwe`

