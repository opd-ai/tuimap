# Getting Started with TuiMap

Welcome to TuiMap! This guide will help you get started with the terminal-based network diagnostic and mapping tool.

## Prerequisites

- Go 1.22 or higher (for building from source)
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

### 3. Edit Configuration (Optional)

Edit `~/.config/tuimap/config.yaml` to customize:
- Scanning methods and timeouts
- Alert rules
- Network interface
- TUI preferences

## Understanding Configuration

### Scanner Settings

```yaml
scanner:
  interface: ""           # Auto-detect if empty
  scan_interval: 60s      # Background scan frequency
  timeout: 10s            # Total scan timeout
  methods:                # Scanning methods to use
    - arp                 # Fastest, local subnet only
    - icmp                # Cross-subnet capability
    - tcp                 # Most reliable through firewalls
```

### Alert Rules

```yaml
alerts:
  enabled: true
  rules:
    - type: new_device
      severity: 1
      action: notify
    - type: device_offline
      severity: 2
      action: log
```

### NAT Settings

```yaml
nat:
  detect: true            # Detect NAT environment
  upnp_enabled: true      # Use UPnP for gateway info
  public_ip_check: true   # Check public IP
```

## Current Status

**Phase 0: Project Setup** ✅ **COMPLETE**

The TuiMap project is currently in Phase 0, which means:
- ✅ Project structure is established
- ✅ CLI framework is working
- ✅ Configuration management is functional
- ⏳ Network scanning is **not yet implemented** (Phase 1)
- ⏳ Device tracking is **not yet implemented** (Phase 2)
- ⏳ Network tools are **not yet implemented** (Phase 3)
- ⏳ Scripting engine is **not yet implemented** (Phase 4)
- ⏳ TUI interface is **not yet implemented** (Phase 5)

## What Works Now

Currently, you can:
- ✅ Run `tuimap --help` to see available commands
- ✅ Run `tuimap version` to check version info
- ✅ Run `tuimap config init` to create configuration
- ✅ Run `tuimap config show` to view configuration

## What's Coming Next

**Phase 1: Core Network Scanner** (Weeks 2-3)
- ARP scanning for local devices
- ICMP ping sweep for cross-subnet discovery
- TCP port scanning for service detection
- Multi-method parallel scanning (<10s target)

**Phase 2: Device Tracking** (Week 4)
- Real-time device status tracking
- Alert engine for new devices and changes
- Persistent device history

**Phase 3: Network Tools** (Weeks 5-6)
- Integrated netcat, telnet, traceroute, dig, whois

**Phase 4: Scripting Engine** (Week 7)
- d5/tengo embedded scripting
- Automation capabilities

**Phase 5: TUI Interface** (Weeks 8-9)
- Interactive terminal UI with multiple views
- Network map visualization
- Device list and details

See [PLAN.md](../PLAN.md) for the complete roadmap.

## Development Roadmap

For detailed implementation plans and progress tracking, see:
- [PLAN.md](../PLAN.md) - Complete implementation plan
- [README.md](../README.md) - Project overview

## Need Help?

- Check the [PLAN.md](../PLAN.md) for architecture details
- Review configuration in `~/.config/tuimap/config.yaml`
- Run `tuimap --help` for command reference

## Contributing

We welcome contributions! The project is in early stages, so there's plenty of opportunity to get involved:

1. Review the [PLAN.md](../PLAN.md) to understand the architecture
2. Pick a phase or component to work on
3. Submit a pull request

## License

TuiMap is licensed under the MIT License. See [LICENSE](../LICENSE) for details.
