# Project Overview

TuiMap is a terminal-based network diagnostic and mapping tool built in Go, designed for real-time network analysis with an emphasis on speed and accuracy in NAT environments. The project aims to discover all devices on local network(s) in under 10 seconds using parallel multi-method scanning (ARP, ICMP, TCP). It provides a modern TUI interface for network visualization, device tracking with alerts, integrated network tools (netcat, telnet, traceroute, dig, whois), and user-extensible automation through an embedded scripting engine.

Target users include network administrators, security professionals, and developers who need fast, reliable network discovery and diagnostic capabilities directly from the terminal. The tool is optimized for NAT environments and multi-subnet networks, making it particularly useful for troubleshooting home networks, enterprise environments, and cloud infrastructure. TuiMap distinguishes itself through its <10 second scan requirement, real-time TUI interface, and extensible scripting capabilities via d5/tengo.

## Technical Stack

- **Primary Language**: Go (version not specified, modern Go assumed with modules support)
- **TUI Framework**: Bubble Tea (charmbracelet/bubbletea) for terminal user interface with lipgloss for styling
- **Networking Libraries**: 
  - gopacket (github.com/google/gopacket) for packet capture and analysis
  - golang.org/x/net/icmp for ICMP protocol implementation
  - gateway (github.com/jackpal/gateway) for gateway detection
- **Scripting Engine**: d5/tengo (github.com/d5/tengo/v2) for embedded scripting with sandboxed execution
- **Storage**: bbolt (go.etcd.io/bbolt) for persistent device history, alerts, and tool execution logs
- **CLI Framework**: Cobra (github.com/spf13/cobra) for command-line interface with Viper (github.com/spf13/viper) for configuration management
- **Testing**: Go's built-in testing package with target >35% code coverage, table-driven tests for business logic
- **Build/Deploy**: Standard Go build toolchain, Makefile for build automation, CI/CD pipeline planned

## Code Assistance Guidelines

1. **Concurrency Patterns**: Use worker pool patterns with configurable concurrency for network scanning. Default to 256 workers for ARP/ICMP scans and 512 for TCP scans. Always use context with timeout (default 10s for full scans) and implement proper cancellation. Follow the pattern: `ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()`. Use sync.WaitGroup for coordinating goroutines and channels for result collection.

2. **Network Scanning Architecture**: Implement parallel multi-method scanning with ARP (layer 2, fastest, local subnet only), ICMP (layer 3, cross-subnet), and TCP SYN scans (most reliable through firewalls). All scan methods must run simultaneously and aggregate results within the 10-second budget. Use hash-based device identification for deduplication. Implement early exit optimization when 99% confidence is achieved.

3. **Data Structures and State Management**: Use the Device struct as the canonical representation: IP (net.IP), MAC (net.HardwareAddr), Hostname (string), Vendor (string), Ports ([]int), LastSeen/FirstSeen (time.Time), Status (DeviceStatus), Metadata (map[string]interface{}). Maintain thread-safe in-memory device registry using sync.RWMutex. Implement event-driven state changes (new, online, offline, changed) with alert triggers.

4. **TUI Component Development**: Follow Bubble Tea's Elm architecture with Model-Update-View pattern. Implement four main views: Network Map (visual topology with connection lines), Device List (sortable table), Tool View (tab-based tool execution), and Script Console (editor with REPL). Use keybindings: 1-4 for view switching, s for scan, r for refresh, f for filter, t for tools, q for quit. Ensure 30 FPS refresh rate through efficient diff rendering.

5. **Scripting API Surface**: Expose network operations to tengo scripts: scan(), ping(), portScan(), resolve() for network functions; alert(), getDevices() for device management; set(), get() for persistent key-value storage. Implement strict sandboxing with resource limits: max_execution_time (30s), max_memory (50MB). Scripts must not have direct file system or command execution access. Follow the pattern in internal/script/api.go for API exposure.

6. **Error Handling and Privilege Management**: Implement graceful degradation for permission issues. Raw socket operations require root/admin - provide clear error messages and fall back to unprivileged methods when possible. Use build tags for platform-specific implementations. Handle errors explicitly with context: "operation failed: %w" wrapping pattern. Never ignore errors in hot paths or critical operations.

7. **Performance Optimization**: Achieve <10s scan time through parallel execution (ARP: 0-3s, ICMP: 0-4s, TCP: 0-3s overlapped, deduplication: <1s). Use sync.Pool for packet buffer reuse, circular buffers for device history, and lazy loading for vendor database. Profile with pprof before optimizing. Keep memory usage under 100MB for 1000 devices. Avoid regex in hot paths; use efficient parsing.

## Project Context

- **Domain**: Network diagnostics and security monitoring with focus on real-time device discovery, change tracking, and alerting. Core business logic centers around multi-method parallel scanning, device state management with temporal tracking, and event-driven alerting system. Critical concepts include NAT traversal (UPnP/NAT-PMP), MAC vendor identification via OUI database, and network topology visualization.

- **Architecture**: Layered architecture with three tiers: (1) System/Network Interface Layer for raw sockets and OS network APIs, (2) Core Engine Layer containing Network Scanner, Device Tracker, Network Tools, and Script Engine, (3) TUI Layer with Bubble Tea framework managing multiple views. Components communicate through channels and event dispatching. State management uses in-memory registry with bbolt persistence for history.

- **Key Directories**:
  - `cmd/tuimap/`: Main application entry point with CLI setup
  - `internal/scanner/`: Network scanning implementations (ARP, ICMP, TCP, passive discovery)
  - `internal/tracker/`: Device state management, alert engine, history tracking
  - `internal/tools/`: Network tool implementations (netcat, telnet, traceroute, dig, whois)
  - `internal/script/`: Tengo VM integration, API bridge, script management
  - `internal/tui/`: Bubble Tea views and UI components
  - `pkg/api/`: Public API definitions for external integration
  - `scripts/`: Example tengo scripts for automation
  - `docs/`: User and API documentation
  - Configuration: `~/.config/tuimap/config.yaml` for user settings, `~/.local/share/tuimap/tuimap.db` for bbolt database

- **Configuration**: YAML-based configuration with scanner settings (interface, scan_interval, timeout, worker counts), alert rules (new_device, device_offline, port_change), NAT settings (UPnP/NAT-PMP, STUN servers), scripting limits (max_execution_time, max_memory), and TUI preferences (theme, refresh_rate, keybindings). Critical environment requirement: raw socket capabilities (CAP_NET_RAW on Linux) for ARP and ICMP scanning.

## Quality Standards

- **Testing Requirements**: Maintain >35% code coverage using Go's built-in testing package. Write table-driven tests for all business logic functions following Go conventions. Include integration tests for network scanning workflows using mocked interfaces. Implement benchmark tests for scan performance validation (must achieve <10s for /24 network). Use testify/suite for complex test scenarios. Mock network interfaces for unit tests; use real network interfaces only in integration tests with appropriate tagging (`//go:build integration`).

- **Code Review Criteria**: All pull requests must pass CI/CD pipeline including linting (golangci-lint), unit tests, and integration tests. Scan performance benchmarks must meet <10s requirement. No decrease in code coverage allowed. Security review required for scripting API changes and raw socket operations. Performance profiling required for changes to scanner or device tracker. Two approvals required for architectural changes; one approval for feature additions and bug fixes.

- **Documentation Standards**: Update relevant documentation in `docs/` directory for any feature additions. API changes must include godoc comments following Go documentation conventions. User-facing features require updates to user guide. Scripting API changes need corresponding updates to script writing guide and example scripts in `scripts/` directory. Architecture Decision Records (ADRs) required for significant design choices - see PLAN.md Appendix B for format. Keep PLAN.md progress tracking section updated after milestone completion.

- **Security and Safety**: Implement rate limiting in scanners to avoid network DoS (configurable scan aggressiveness). Script sandboxing is mandatory with strict resource limits and no privileged operations. No telemetry or cloud data transmission - all data stays local. Support optional database encryption for sensitive environments. Follow principle of least privilege - request minimal permissions and gracefully degrade when not available. Validate all user inputs in CLI and script APIs.

- **Performance Monitoring**: Track key metrics defined in PLAN.md Section 3.1: scan time (<10s for /24), device detection rate (100%), alert latency (<500ms), UI refresh rate (30 FPS), memory usage (<100MB for 1k devices), startup time (<2s). Run performance benchmarks weekly in CI. Use pprof for CPU and memory profiling during optimization. Implement adaptive scanning that adjusts strategy based on network response patterns and available time budget.
