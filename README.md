# TuiMap

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

TuiMap is a terminal-based network diagnostic and mapping tool built in Go, designed for real-time network analysis with an emphasis on speed and accuracy in NAT environments.

## Features

- 🚀 **Fast Network Scanning** - Discover all devices on /24 networks in under 10 seconds
- 📊 **Real-time Device Tracking** - Monitor device status changes and receive alerts
- 🛠️ **Integrated Network Tools** - Built-in netcat, telnet, traceroute, dig, and whois
- 📜 **Extensible Scripting** - Automate tasks with embedded Tengo scripts
- 🎨 **Modern TUI Interface** - Interactive terminal UI with multiple views
- 🌐 **NAT Environment Support** - Optimized for NAT environments and multi-subnet networks

## Installation

### From Source

```bash
git clone https://github.com/opd-ai/tuimap.git
cd tuimap
make build
sudo make install  # Optional: install to GOPATH/bin
```

### Binary

```bash
./bin/tuimap --help
```

## Quick Start

### Initialize Configuration

```bash
tuimap config init
```

This creates a default configuration file at `~/.config/tuimap/config.yaml`.

### View Current Configuration

```bash
tuimap config show
```

### Run TuiMap

```bash
# Run with default settings
sudo tuimap

# Run with specific interface
sudo tuimap --interface eth0

# Run in debug mode
sudo tuimap --debug
```

**Note:** Root/admin privileges are required for raw socket operations (ARP and ICMP scanning). The tool will gracefully degrade to unprivileged methods when run without elevated permissions.

## Configuration

Configuration file location: `~/.config/tuimap/config.yaml`

Key configuration sections:
- **scanner** - Network scanning settings (methods, timeouts, worker counts)
- **alerts** - Alert rules and notifications
- **nat** - NAT detection and traversal settings
- **scripting** - Script engine configuration
- **tui** - Terminal UI preferences
- **storage** - Database and history settings
- **logging** - Log level and file settings

See the generated config file for detailed options and defaults.

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile)

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Format code
make fmt

# Run linter (if golangci-lint is installed)
make lint

# Clean build artifacts
make clean
```

### Project Structure

```
tuimap/
├── cmd/tuimap/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── scanner/         # Network scanning (ARP, ICMP, TCP)
│   ├── tracker/         # Device state management
│   ├── tools/           # Network diagnostic tools
│   ├── script/          # Tengo scripting engine
│   └── tui/             # Terminal UI (Bubble Tea)
├── pkg/api/             # Public API interfaces
├── scripts/             # Example Tengo scripts
├── docs/                # Documentation
├── Makefile             # Build automation
├── go.mod               # Go module definition
└── PLAN.md              # Implementation roadmap
```

## Current Status

**Phase 0: Project Setup - ✅ COMPLETE**

The foundational structure is in place:
- ✅ Go module initialized
- ✅ Directory structure created
- ✅ CLI framework with Cobra
- ✅ Configuration management with Viper
- ✅ Build system with Makefile
- ✅ Core package interfaces defined

**Next Phase: Phase 1 - Core Network Scanner**

See [PLAN.md](PLAN.md) for the complete implementation roadmap.

## Known Limitations

- **NAT Port Mapping**: NAT detection (type detection, public IP via STUN, UPnP/NAT-PMP discovery) is fully functional. However, NAT port mapping is not yet fully implemented. `AddPortMapping()` currently reports port-mapping failures rather than returning `ErrNATUnsupported`, and `RemovePortMapping()` currently performs no action and may return `nil`. NAT port mapping requires external gateway support and may be completed in a future release.
- **Root/Admin Privileges**: ARP and ICMP scanning require raw socket access (root on Linux, admin on Windows). The tool gracefully degrades to TCP-only scanning when run without elevated permissions.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Roadmap

- **Phase 1** (Weeks 2-3): Core Network Scanner (ARP, ICMP, TCP scanning)
- **Phase 2** (Week 4): Device Tracking System
- **Phase 3** (Weeks 5-6): Network Tools Integration
- **Phase 4** (Week 7): Scripting Engine (d5/tengo)
- **Phase 5** (Weeks 8-9): TUI Implementation (Bubble Tea)
- **Phase 6** (Week 10): NAT & Advanced Features
- **Phase 7** (Weeks 11-12): Testing & Documentation

See [PLAN.md](PLAN.md) for detailed milestones and progress tracking.

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Tengo](https://github.com/d5/tengo) - Embedded scripting
- [gopacket](https://github.com/gopacket/gopacket) - Packet capture
