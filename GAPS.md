# Implementation Gaps — 2026-04-07

This document details the gaps between TuiMap's stated goals (from README.md and PLAN.md) and its current implementation status. Each gap represents a feature that is documented/promised but not yet functional.

---

## Network Scanner (CRITICAL)

- **Stated Goal**: "Discover all devices on /24 networks in under 10 seconds" using parallel multi-method scanning (ARP, ICMP, TCP)
- **Current State**: Only interface definition exists at `internal/scanner/scanner.go:50-57`. Lines 59-63 contain TODO comments for ARP, ICMP, TCP, and passive discovery scanners. No actual scanning implementation. Zero dependencies for packet capture (gopacket not in go.mod).
- **Impact**: This is the project's core value proposition. Without functional scanning, TuiMap provides no network discovery capability whatsoever. Users cannot discover devices, making the tool unusable for its primary purpose.
- **Closing the Gap**:
  1. Add dependencies: `go get github.com/google/gopacket golang.org/x/net/icmp github.com/jackpal/gateway`
  2. Implement `internal/scanner/arp.go` — ARP scanner with 256 worker goroutines, 100ms timeout per request
  3. Implement `internal/scanner/icmp.go` — ICMP ping sweep with 256 workers, 1s timeout
  4. Implement `internal/scanner/tcp.go` — TCP SYN scanner on ports 22,80,443,3389,5900 with 512 workers
  5. Implement `internal/scanner/orchestrator.go` — Parallel execution of all methods with 10s total timeout
  6. Create benchmark tests validating <10s scan time for /24 network

---

## Device Tracker (CRITICAL)

- **Stated Goal**: "Monitor device status changes and receive alerts" with real-time device state management, alert engine, and persistent history
- **Current State**: Only types and interface defined at `internal/tracker/tracker.go`. Tracker interface (lines 37-50) has no implementations. TODO comments at lines 52-54 mark unimplemented features. No in-memory registry, alert logic, or persistence layer.
- **Impact**: Users cannot track device changes over time or receive notifications for new devices, offline events, or port changes. Network monitoring use cases are completely unsupported.
- **Closing the Gap**:
  1. Add dependency: `go get go.etcd.io/bbolt`
  2. Implement `internal/tracker/registry.go` — Thread-safe in-memory device map with `sync.RWMutex`
  3. Implement `internal/tracker/state.go` — State change detection (new, online, offline, changed)
  4. Implement `internal/tracker/alerts.go` — Rule-based alert engine matching config.AlertsConfig
  5. Implement `internal/tracker/storage.go` — bbolt persistence for device history
  6. Wire tracker to receive scan results and emit events via channels

---

## Network Tools — Netcat, Telnet, Traceroute, Dig, Whois (CRITICAL)

- **Stated Goal**: "Built-in netcat, telnet, traceroute, dig, and whois" as integrated network diagnostic tools
- **Current State**: Only NetworkTool interface exists at `internal/tools/tools.go:9-19`. Lines 21-25 contain TODO comments for all 5 tools. Zero implementations.
- **Impact**: Users cannot perform any network diagnostics within TuiMap. The integrated tools feature is entirely non-functional. Users must exit TuiMap to use standard system utilities.
- **Closing the Gap**:
  1. Implement `internal/tools/netcat.go` — TCP/UDP client with send/receive, optional listen mode
  2. Implement `internal/tools/telnet.go` — Telnet protocol with WILL/WONT/DO/DONT negotiation
  3. Implement `internal/tools/traceroute.go` — ICMP or UDP-based path discovery with hop timing
  4. Implement `internal/tools/dig.go` — DNS queries (A, AAAA, MX, TXT, NS, CNAME) using `net.Resolver`
  5. Implement `internal/tools/whois.go` — RFC 3912 WHOIS client for domain/IP lookups
  6. Create base framework for tool lifecycle management and output streaming

---

## TUI Interface (CRITICAL)

- **Stated Goal**: "Interactive terminal UI with multiple views" including Network Map View, Device List View, Tool View, and Script Console per PLAN.md Section 1.2.5
- **Current State**: Package `internal/tui/tui.go` contains only package declaration and TODO comments (lines 5-9). No Bubble Tea integration, no views, no models, no keybindings.
- **Impact**: Users have no interactive interface. The application only shows CLI help text. All TUI-related claims in README are non-functional.
- **Closing the Gap**:
  1. Add dependencies: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss`
  2. Implement `internal/tui/app.go` — Main Bubble Tea Model with view switching (keys 1-4)
  3. Implement `internal/tui/views/network_map.go` — Visual topology with gateway relationships
  4. Implement `internal/tui/views/device_list.go` — Sortable table with filter/search
  5. Implement `internal/tui/views/tool_view.go` — Tab interface for network tools
  6. Implement `internal/tui/views/script_console.go` — Script editor and execution display
  7. Add keybindings: q=quit, s=scan, r=refresh, f=filter, t=tools

---

## Scripting Engine (CRITICAL)

- **Stated Goal**: "Automate tasks with embedded Tengo scripts (d5/tengo)" per README, with API for network operations, alerts, and persistent storage per PLAN.md Section 1.2.4
- **Current State**: Only Engine interface at `internal/script/script.go:9-19`. d5/tengo is not in go.mod. Example script at `scripts/examples/auto-scan.tengo` explicitly states "scripting engine is not yet implemented".
- **Impact**: Users cannot automate any network tasks. The extensibility claim is non-functional. Custom monitoring and alerting scripts cannot be created.
- **Closing the Gap**:
  1. Add dependency: `go get github.com/d5/tengo/v2`
  2. Implement `internal/script/engine.go` — Tengo VM initialization with resource limits
  3. Implement `internal/script/api.go` — Expose scan(), ping(), portScan(), resolve(), alert(), getDevices(), set(), get() functions
  4. Implement `internal/script/sandbox.go` — Resource limits (30s execution, 50MB memory), no file system access
  5. Implement script hot-reload and management
  6. Validate example scripts execute correctly

---

## NAT Environment Support (HIGH)

- **Stated Goal**: "Optimized for NAT environments and multi-subnet networks" with UPnP/NAT-PMP integration and STUN client per PLAN.md Section 1.2.1
- **Current State**: Config struct defines NATConfig at `internal/config/config.go:69-75` but no implementation exists. No UPnP, NAT-PMP, STUN, or gateway detection code anywhere in codebase.
- **Impact**: Users in NAT environments (home routers, enterprise NAT, cloud VPCs) cannot discover external IP addresses, traverse NAT for remote scanning, or detect multi-homed network topology.
- **Closing the Gap**:
  1. Create `internal/nat/` package
  2. Implement `internal/nat/upnp.go` — SSDP discovery and UPnP IGD port mapping queries
  3. Implement `internal/nat/natpmp.go` — NAT-PMP client for compatible routers
  4. Implement `internal/nat/stun.go` — STUN client using servers from config (stun.l.google.com:19302)
  5. Implement `internal/scanner/gateway.go` — Gateway detection using `github.com/jackpal/gateway`
  6. Integrate NAT detection with scanner for adaptive scanning strategies

---

## Test Coverage (HIGH)

- **Stated Goal**: ">80% test coverage" per PLAN.md Section 4.5 quality requirements
- **Current State**: All 8 packages report "no test files" when running `go test ./...`. Test coverage is 0%.
- **Impact**: Code quality cannot be validated. Regressions will go undetected. Contributors cannot verify their changes don't break existing functionality.
- **Closing the Gap**:
  1. Create `internal/config/config_test.go` — Test LoadConfig, InitConfig, DefaultConfig
  2. Create test files for each package as features are implemented
  3. Use table-driven tests following Go conventions
  4. Add mock interfaces for network operations in tests
  5. Configure CI to enforce coverage threshold: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`

---

## Public API (HIGH)

- **Stated Goal**: `pkg/api/` package for "public API definitions for external integration" per project structure in README
- **Current State**: `pkg/api/api.go` contains only "TODO: Define public API interfaces" comment (line 4)
- **Impact**: External consumers cannot programmatically use TuiMap as a library. Only CLI usage is available.
- **Closing the Gap**:
  1. Define exported interfaces mirroring internal capabilities
  2. Export: Scanner, Tracker, Device, Alert, NetworkTool, ScriptEngine types
  3. Provide factory functions: NewScanner(), NewTracker(), etc.
  4. Add comprehensive godoc documentation
  5. Create usage examples in `pkg/api/example_test.go`

---

## CI/CD Pipeline (MEDIUM)

- **Stated Goal**: Automated testing and linting per Makefile targets and PLAN.md Phase 7
- **Current State**: `.github/workflows/` directory exists but contains no workflow files. No automated builds on push/PR.
- **Impact**: Code quality is not automatically validated. PRs can merge with failing tests or linting errors.
- **Closing the Gap**:
  1. Create `.github/workflows/ci.yml` with:
     - `go test -race ./...`
     - `go vet ./...`
     - `golangci-lint run ./...`
     - Coverage reporting to Codecov or similar
  2. Add branch protection rules requiring CI pass
  3. Add status badges to README.md

---

## CLI Default Behavior (MEDIUM)

- **Stated Goal**: Running `tuimap` should launch the interactive TUI per README usage examples
- **Current State**: `cmd/tuimap/main.go:31-34` — rootCmd.Run only calls `cmd.Help()`. No TUI launch, no scanning.
- **Impact**: Users expecting a network scanner get only help text. The default experience doesn't demonstrate any core functionality.
- **Closing the Gap**:
  1. Once TUI is implemented, modify rootCmd.Run to launch Bubble Tea application
  2. Add `--no-tui` flag for headless/batch mode
  3. Add `scan` subcommand for one-shot scanning without TUI

---

## Config Show Command (LOW)

- **Stated Goal**: `tuimap config show` should display current configuration
- **Current State**: `cmd/tuimap/main.go:74` has TODO comment. Command only shows config file path, not contents.
- **Impact**: Users cannot inspect their configuration without manually reading the YAML file.
- **Closing the Gap**:
  1. Marshal config struct to YAML and print to stdout
  2. Add `--json` flag for machine-readable output
  3. Consider `--get KEY` for querying specific values

---

## Summary

| Gap | Severity | Effort Estimate | Dependencies |
|-----|----------|-----------------|--------------|
| Network Scanner | CRITICAL | 2 weeks | gopacket, icmp, gateway |
| Device Tracker | CRITICAL | 1 week | bbolt, scanner complete |
| Network Tools (5) | CRITICAL | 1.5 weeks | None |
| TUI Interface | CRITICAL | 2 weeks | bubbletea, scanner/tracker complete |
| Scripting Engine | CRITICAL | 1 week | tengo, scanner/tracker complete |
| NAT Support | HIGH | 1 week | gateway, scanner complete |
| Test Coverage | HIGH | Ongoing | Each feature implementation |
| Public API | HIGH | 3 days | Core features complete |
| CI/CD Pipeline | MEDIUM | 2 days | None |
| CLI Default Behavior | MEDIUM | 1 day | TUI complete |
| Config Show | LOW | 2 hours | None |

**Total estimated effort to close all gaps: 8-9 weeks** (aligns with PLAN.md 12-week timeline minus completed Phase 0)

---

*Gaps analysis generated 2026-04-07 based on comparison of README.md/PLAN.md claims vs. actual codebase implementation*
