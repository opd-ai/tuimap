# Goal-Achievement Assessment

## Project Context

### What it claims to do
TuiMap is a terminal-based network diagnostic and mapping tool designed for real-time network analysis with emphasis on speed and accuracy in NAT environments. Key claims from README and PLAN.md:

1. **Fast Network Scanning** - Discover all devices on /24 networks in under 10 seconds
2. **Real-time Device Tracking** - Monitor device status changes and receive alerts
3. **Integrated Network Tools** - Built-in netcat, telnet, traceroute, dig, and whois
4. **Extensible Scripting** - Automate tasks with embedded Tengo scripts (d5/tengo)
5. **Modern TUI Interface** - Interactive terminal UI with multiple views (Network Map, Device List, Tool View, Script Console)
6. **NAT Environment Support** - Optimized for NAT environments and multi-subnet networks

### Target Audience
- Network administrators
- Security professionals
- Developers needing fast, reliable network discovery and diagnostics from the terminal
- Users managing home networks, enterprise environments, and cloud infrastructure

### Architecture
| Package | Role | Status |
|---------|------|--------|
| `cmd/tuimap` | CLI entry point with Cobra | ✅ Implemented |
| `internal/config` | Configuration management with Viper | ✅ Implemented |
| `internal/scanner` | Network scanning (ARP, ICMP, TCP) | ❌ Interface only |
| `internal/tracker` | Device state management and alerts | ❌ Interface only |
| `internal/tools` | Network diagnostic tools | ❌ Interface only |
| `internal/script` | Tengo scripting engine | ❌ Interface only |
| `internal/tui` | Bubble Tea terminal UI | ❌ Interface only |
| `pkg/api` | Public API definitions | ❌ Empty |

### Existing CI/Quality Gates
- **Makefile targets**: `build`, `test`, `fmt`, `vet`, `lint`, `test-coverage`
- **Test command**: `go test -v -race -coverprofile=coverage.out ./...`
- **Linting**: golangci-lint (optional)
- **No CI pipeline configured** (`.github/workflows/` is empty)

---

## Goal-Achievement Summary

| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| Fast Network Scanning (<10s) | ❌ Missing | `internal/scanner/scanner.go` contains only interface definitions and TODO comments (lines 59-63) | No ARP, ICMP, or TCP scanner implementations exist |
| Real-time Device Tracking | ❌ Missing | `internal/tracker/tracker.go` contains only types and interface; TODO comments at lines 52-54 | No tracker implementation, no alert engine, no persistence |
| Integrated Network Tools | ❌ Missing | `internal/tools/tools.go` has interface only; TODO at lines 21-25 | None of the 5 tools (netcat, telnet, traceroute, dig, whois) implemented |
| Extensible Scripting | ❌ Missing | `internal/script/script.go` has interface only; TODO at lines 21-23 | No Tengo VM integration, no API bridge |
| Modern TUI Interface | ❌ Missing | `internal/tui/tui.go` has only TODO comments (lines 5-9) | No Bubble Tea views implemented |
| NAT Environment Support | ❌ Missing | No NAT-related code exists | UPnP/NAT-PMP, gateway detection, STUN not implemented |
| CLI Framework | ✅ Achieved | `cmd/tuimap/main.go` - Cobra CLI with version, config init/show commands | Working basic CLI |
| Configuration Management | ✅ Achieved | `internal/config/config.go` - 258 lines, comprehensive config struct with Viper | Full config schema with sensible defaults |
| >80% Test Coverage | ❌ Missing | `go test ./...` reports "no test files" for all packages | 0% test coverage |
| Cross-platform Support | ⚠️ Partial | Go code compiles; no platform-specific implementations | No build tags or platform-specific code |

**Overall: 2/10 goals achieved** (CLI framework and configuration only)

---

## Metrics Summary

From `go-stats-generator` analysis:

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Lines of Code | 126 | Very early stage |
| Total Functions | 5 | Minimal implementation |
| Total Structs | 17 | Good type definitions |
| Total Interfaces | 4 | Scanner, Tracker, NetworkTool, Engine defined |
| Total Packages | 8 | Architecture laid out |
| Documentation Coverage | 46.2% | Type coverage at 39.1% |
| TODO Comments | 22 | All core features marked as TODO |
| Test Files | 0 | No tests exist |
| Cyclomatic Complexity | Max 5 (LoadConfig) | Acceptable |

---

## Roadmap

### Priority 1: Implement Core Network Scanner (Critical)

The <10s scanning requirement is the project's primary differentiator. Without it, TuiMap has no value proposition over existing tools like nmap or Angry IP Scanner.

**Implementation Steps:**

- [ ] **ARP Scanner** (`internal/scanner/arp.go`)
  - Implement raw socket ARP requests with gopacket
  - Worker pool pattern (256 concurrent workers as per PLAN.md)
  - Subnet detection and CIDR calculation
  - MAC vendor lookup via OUI database
  - Target: <3s for /24 network
  - Reference: PLAN.md Section 3.2.1

- [ ] **ICMP Scanner** (`internal/scanner/icmp.go`)
  - Use `golang.org/x/net/icmp` for ping sweep
  - 256 concurrent workers with 1s timeout per host
  - Graceful fallback when raw sockets unavailable
  - Target: <4s for /24 network
  - Reference: PLAN.md Section 3.2.2

- [ ] **TCP Port Scanner** (`internal/scanner/tcp.go`)
  - TCP SYN scan on configurable ports (default: 22, 80, 443, 3389, 5900)
  - 512 concurrent workers, 500ms timeout per connection
  - Service/banner detection
  - Target: <3s for /24 network with 5 ports
  - Reference: PLAN.md Section 3.2.3

- [ ] **Multi-method Orchestrator** (`internal/scanner/orchestrator.go`)
  - Run all scan methods in parallel with `context.WithTimeout(10s)`
  - Hash-based device deduplication
  - Result aggregation and merging
  - Early exit when 99% confidence achieved
  - Reference: PLAN.md Section 3.6.1

- [ ] **Add dependencies to go.mod**
  - `github.com/google/gopacket` for packet capture
  - `golang.org/x/net/icmp` for ICMP
  - `github.com/jackpal/gateway` for gateway detection

**Validation:**
```bash
# Benchmark scan time on /24 network
go test -bench=BenchmarkScan ./internal/scanner/... -benchtime=5x
# Must complete in <10s average
```

---

### Priority 2: Implement Device Tracker and Alerts

Real-time device tracking is the second core feature enabling monitoring use cases.

- [ ] **In-memory Device Registry** (`internal/tracker/registry.go`)
  - Thread-safe map with `sync.RWMutex`
  - Device struct: IP, MAC, Hostname, Vendor, Ports, LastSeen, FirstSeen, Status
  - LRU eviction for memory efficiency

- [ ] **State Change Detection** (`internal/tracker/state.go`)
  - Detect: new device, online, offline, changed (ports/MAC)
  - Event channels for subscribers
  - Configurable offline threshold

- [ ] **Alert Engine** (`internal/tracker/alerts.go`)
  - Rule-based alerts matching config.AlertsConfig
  - <500ms latency requirement
  - Alert types: new_device, device_offline, port_change, mac_conflict

- [ ] **Persistence Layer** (`internal/tracker/storage.go`)
  - Add `go.etcd.io/bbolt` to go.mod
  - Implement schema from PLAN.md Section 3.3.1
  - History tracking with configurable retention

**Validation:**
```bash
go test -race ./internal/tracker/...
# Alert latency test must show <500ms
```

---

### Priority 3: Implement TUI Interface

The TUI is essential for the "interactive terminal UI" claim and user experience.

- [ ] **Add Bubble Tea dependencies**
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/lipgloss`

- [ ] **Core TUI Framework** (`internal/tui/app.go`)
  - Bubble Tea Model-Update-View pattern
  - View switching logic (keys 1-4)
  - Global keybindings (q=quit, s=scan, r=refresh)

- [ ] **Network Map View** (`internal/tui/views/network_map.go`)
  - Visual topology with gateway relationships
  - Real-time status indicators (online/offline)
  - Device selection and navigation

- [ ] **Device List View** (`internal/tui/views/device_list.go`)
  - Sortable table (IP, MAC, Hostname, Vendor, Status, Last Seen)
  - Filter/search with 'f' key
  - Detail pane on selection

- [ ] **Tool View** (`internal/tui/views/tool_view.go`)
  - Tab interface for each tool
  - Input area and scrollable output
  - Command history

- [ ] **Script Console** (`internal/tui/views/script_console.go`)
  - Script file loader
  - Execution controls and output display

**Validation:**
```bash
# Manual testing on 80x24 terminal
# Must maintain 30 FPS with no flickering
```

---

### Priority 4: Implement Network Tools

Five integrated tools are explicitly promised in the README.

- [ ] **Tool Framework** (`internal/tools/base.go`)
  - Common interface implementation
  - Context cancellation support
  - Output streaming via channels

- [ ] **Netcat** (`internal/tools/netcat.go`)
  - TCP/UDP client mode
  - Optional listen mode

- [ ] **Telnet** (`internal/tools/telnet.go`)
  - Telnet protocol with WILL/WONT/DO/DONT negotiation
  - Banner grabbing

- [ ] **Traceroute** (`internal/tools/traceroute.go`)
  - ICMP or UDP-based
  - Hop timing display

- [ ] **Dig** (`internal/tools/dig.go`)
  - DNS queries (A, AAAA, MX, TXT, NS, CNAME)
  - Custom resolver support

- [ ] **Whois** (`internal/tools/whois.go`)
  - WHOIS protocol client
  - Domain and IP lookups

**Validation:**
```bash
# Integration tests for each tool
go test ./internal/tools/... -tags=integration
```

---

### Priority 5: Implement Scripting Engine

Extensible scripting is a key differentiator from nmap/similar tools.

- [ ] **Add Tengo dependency**
  - `github.com/d5/tengo/v2`

- [ ] **Tengo VM Integration** (`internal/script/engine.go`)
  - VM initialization with resource limits (30s execution, 50MB memory)
  - Script loading from file and string
  - Hot reload support

- [ ] **API Bridge** (`internal/script/api.go`)
  - Expose scan(), ping(), portScan(), resolve()
  - Expose alert(), getDevices(), getDevice()
  - Expose set(), get(), delete() for persistent storage
  - Reference: PLAN.md Section 3.5

- [ ] **Sandboxing** (`internal/script/sandbox.go`)
  - No direct file system access
  - No command execution
  - Whitelist of allowed operations

**Validation:**
```bash
# Run example script
go test ./internal/script/... -run TestExampleScript
```

---

### Priority 6: NAT Environment Support

NAT optimization is explicitly claimed and differentiates from basic scanners.

- [ ] **Gateway Detection** (`internal/scanner/gateway.go`)
  - Use `github.com/jackpal/gateway` for default gateway
  - Parse routing tables for multi-homed systems

- [ ] **UPnP/NAT-PMP Integration** (`internal/nat/upnp.go`)
  - Discover router capabilities
  - Query external IP address

- [ ] **STUN Client** (`internal/nat/stun.go`)
  - Public IP discovery via STUN servers
  - NAT type detection

- [ ] **Multi-subnet Support** (`internal/scanner/multisubnet.go`)
  - Route table parsing
  - Cross-subnet discovery strategies

---

### Priority 7: Testing and CI/CD

>80% test coverage is explicitly required in PLAN.md Section 4.5.

- [ ] **Unit Tests**
  - `internal/config/config_test.go` - config loading/saving
  - `internal/scanner/*_test.go` - mock network interfaces
  - `internal/tracker/*_test.go` - state management
  - `internal/tools/*_test.go` - tool execution

- [ ] **Integration Tests** (build tag: `// +build integration`)
  - End-to-end scan workflows
  - Real network interface tests

- [ ] **Benchmark Tests**
  - `BenchmarkARPScan`, `BenchmarkICMPScan`, `BenchmarkTCPScan`
  - Must validate <10s requirement

- [ ] **CI Pipeline** (`.github/workflows/ci.yml`)
  - Run tests on push/PR
  - golangci-lint
  - Coverage reporting

**Validation:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
# Must show >80%
```

---

## Implementation Order Rationale

1. **Scanner first** - Core differentiator (<10s claim), enables all other features
2. **Tracker second** - Enables alerting and persistent state, required for TUI data
3. **TUI third** - Primary user interface, depends on scanner/tracker data
4. **Tools fourth** - Standalone features, lower priority than core scanning
5. **Scripting fifth** - Advanced feature, requires stable core APIs
6. **NAT sixth** - Enhancement to scanning, not blocking for basic functionality
7. **Testing last** - Should actually be done alongside each phase, listed last for roadmap clarity

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Raw socket permissions | Implement unprivileged fallback (TCP connect scan, external ping) |
| Scan time >10s | Early exit optimization, adaptive worker pools, profile with pprof |
| gopacket complexity | Consider libpcap-less pure Go alternatives for ARP |
| Cross-platform | Use build tags (`// +build linux darwin windows`) for platform-specific code |

---

## Estimated Effort

Based on PLAN.md timeline and current state:

| Priority | Estimated Duration | Dependencies |
|----------|-------------------|--------------|
| P1: Scanner | 2 weeks | go.mod updates |
| P2: Tracker | 1 week | P1 complete |
| P3: TUI | 2 weeks | P1, P2 complete |
| P4: Tools | 1.5 weeks | None |
| P5: Scripting | 1 week | P1, P2 complete |
| P6: NAT | 1 week | P1 complete |
| P7: Testing | Ongoing | All phases |

**Total: ~8-9 weeks to v1.0** (aligns with PLAN.md 12-week estimate minus already-complete Phase 0)

---

## Cleanup Note

The following temporary file was created during analysis and has been cleaned up:
- `/tmp/review-metrics.json` - Deleted after use
