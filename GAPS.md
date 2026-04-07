# Implementation Gaps — 2026-04-07

This document identifies gaps between TuiMap's stated goals and its current implementation.

---

## 1. Test Coverage Target Not Met

- **Stated Goal**: >35% test coverage (from project conventions and quality standards)
- **Current State**: Overall coverage approximately 52%. Individual packages range from 14.8% (`internal/tools`) to 91.4% (`internal/tui`)
- **Impact**: Reduced confidence in code correctness. The scanner and tools packages—core functionality—have the lowest coverage, increasing risk of undetected regressions
- **Closing the Gap**:
  1. **internal/tools** (14.8% → 35%): Add tests for netcat, telnet, traceroute, whois. Mock network connections for deterministic tests. Focus on `Execute()` methods and edge cases. Estimated: 2 days
  2. **internal/scanner** (37.9% → 35%): Add tests for ARP, ICMP, TCP scanners. Use mock interfaces. Test `Scan()`, `pingWorker()`, `mergeDevices()` which have highest complexity. Estimated: 3 days  
  3. **internal/config** (48.8% → 35%): Test `LoadConfig()` with missing file, malformed YAML, env var overrides. Estimated: 1 day
  4. **internal/script** (54.5% → 35%): Test timeout behavior, memory limits, invalid scripts, all API functions. Estimated: 1.5 days
  5. **internal/nat** (67.3% → 35%): Mock STUN responses, test UPnP/NAT-PMP discovery edge cases. Estimated: 1 day

  **Validation**:
  ```bash
  go test -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out | grep total
  # Must show ≥35%
  ```

---

## 2. UPnP/NAT-PMP Port Mapping (Stub Implementation)

- **Stated Goal**: NAT package provides `AddPortMapping()` and `RemovePortMapping()` via `NATClient` interface (documented in `pkg/api/api.go` and `internal/nat/nat.go:75-78`)
- **Current State**: Methods `addMappingUPnP()` and `addMappingNATPMP()` at `internal/nat/nat.go:501-512` return `ErrNATUnsupported`. The public `AddPortMapping()` method at line 176 calls these stubs and always fails
- **Impact**: Users cannot programmatically create port forwarding rules despite the interface promising this capability. This affects scenarios where TuiMap needs to make services accessible across NAT boundaries
- **Closing the Gap**:
  1. **Option A (Full Implementation)**: Implement UPnP IGD SOAP calls for `AddPortMapping` action. Parse SSDP discovery response to get control URL, then make SOAP POST to `urn:schemas-upnp-org:service:WANIPConnection:1`. For NAT-PMP, send proper mapping request packets per RFC 6886
  2. **Option B (Remove from Interface)**: Remove `AddPortMapping()` and `RemovePortMapping()` from `NATClient` interface if not planned for implementation. Update documentation to clarify port mapping is not supported
  3. **Option C (Document Limitation)**: Keep interface but document that port mapping requires external gateway support and returns `ErrNATUnsupported` when not available

  **Validation**:
  ```bash
  # If implementing:
  go test ./internal/nat/... -v -run TestAddPortMapping
  # If removing:
  grep -n "AddPortMapping" pkg/api/*.go internal/nat/*.go
  # Should show no references after removal
  ```

---

## 3. Scan Performance Benchmark Validation

- **Stated Goal**: "Discover all devices on /24 networks in under 10 seconds" (README.md line 10)
- **Current State**: Implementation exists with 10-second timeout in `CreateDefaultOrchestrator()` (`internal/scanner/orchestrator.go:234`). However, there are no benchmark tests validating this claim
- **Impact**: The core differentiating claim cannot be verified. Performance regressions could go undetected
- **Closing the Gap**:
  1. Create `internal/scanner/benchmark_test.go` with benchmark functions:
     - `BenchmarkARPScan` — target <3s for /24
     - `BenchmarkICMPScan` — target <4s for /24
     - `BenchmarkTCPScan` — target <3s for /24 (5 ports)
     - `BenchmarkOrchestrator` — target <10s end-to-end
  2. Add CI step to run benchmarks weekly or on performance-critical PRs
  3. Implement benchmark comparison to detect regressions

  **Validation**:
  ```bash
  go test -bench=BenchmarkOrchestrator ./internal/scanner/... -benchtime=5x
  # Average time must be <10s
  ```

---

## 4. TUI Scan Integration Incomplete

- **Stated Goal**: Modern TUI interface with network scanning capability (pressing 's' initiates scan)
- **Current State**: The TUI at `internal/tui/app.go:149-150` handles 's' key by setting status to "Scanning..." but does not actually invoke any scanner. The `Model` struct has `scanResult` field but no code populates it. `SetDevices()` exists at line 348 but is never called from the Update loop
- **Impact**: Users see "Scanning..." status but no scan occurs. The TUI is display-only without actual network discovery integration
- **Closing the Gap**:
  1. Add `scanner.Orchestrator` and `tracker.Registry` fields to `Model` struct
  2. In 's' key handler, send a `tea.Cmd` that invokes `orchestrator.Scan()`
  3. Create message type for scan results and update `Model.devices` in Update
  4. Wire `registry.Update()` to process scan results and generate alerts
  5. Add refresh timer cmd for periodic scanning based on config

  **Validation**:
  ```bash
  # Build and run TUI
  go build ./cmd/tuimap && sudo ./tuimap
  # Press 's' - should see devices populate in Device List view
  ```

---

## 5. Storage Layer Not Integrated

- **Stated Goal**: Persistent device history using bbolt database (README mentions `~/.local/share/tuimap/tuimap.db`)
- **Current State**: `internal/tracker/storage.go` implements `Storage` type with `SaveDevice()`, `LoadDevices()`, etc. However:
  - `cmd/tuimap/main.go` does not instantiate Storage
  - TUI does not call storage methods
  - No persistence actually occurs
- **Impact**: Device history is lost between sessions. History retention setting in config has no effect
- **Closing the Gap**:
  1. In `cmd/tuimap/main.go`, create Storage instance from config
  2. Pass Storage to Registry or create composite type
  3. Call `Storage.SaveDevices()` after each scan
  4. Call `Storage.LoadDevices()` on startup to restore state
  5. Implement history cleanup based on `storage.history_retention` config

  **Validation**:
  ```bash
  # Run scan, quit, restart - previous devices should appear
  sudo ./tuimap
  # Press 's' to scan, 'q' to quit
  sudo ./tuimap
  # Devices from previous session should be visible
  ```

---

## 6. CLI Scan Command Missing

- **Stated Goal**: Documentation shows `tuimap scan --subnet 192.168.1.0/24` (USER_GUIDE.md lines 77-88)
- **Current State**: `cmd/tuimap/main.go` only defines `rootCmd` (launches TUI), `versionCmd`, `configCmd`, `configInitCmd`, and `configShowCmd`. There is no `scan` subcommand
- **Impact**: Users cannot perform headless scans from command line as documented. The `--no-tui` flag only shows help text
- **Closing the Gap**:
  1. Add `scanCmd` with flags: `--subnet`, `--interface`, `--methods`, `--timeout`
  2. Implement headless scan that outputs results to stdout (JSON or table format)
  3. Support `--output json` for scriptable output
  4. Integrate with `--no-tui` flag behavior

  **Validation**:
  ```bash
  sudo ./tuimap scan --subnet 192.168.1.0/24 --output json
  # Should output JSON array of discovered devices
  ```

---

## 7. Multi-Subnet Scanning Not Exposed

- **Stated Goal**: "TuiMap can discover and scan multiple subnets" with `--all-subnets` and `--from-routes` flags (USER_GUIDE.md lines 103-110)
- **Current State**: `internal/scanner/multisubnet.go` implements `MultiSubnetScanner`, `DiscoverSubnets()`, and `ParseRoutingTable()`. However, these are not exposed via CLI or TUI
- **Impact**: Users cannot utilize multi-subnet scanning capability despite implementation existing
- **Closing the Gap**:
  1. Add `--all-subnets` flag to scan command that calls `DiscoverSubnets()`
  2. Add `--from-routes` flag that calls `ParseRoutingTable()`
  3. In TUI, add subnet selector or auto-detect option
  4. Update orchestrator to handle multiple subnet results

  **Validation**:
  ```bash
  sudo ./tuimap scan --all-subnets
  # Should discover and scan all local subnets
  ```

---

## 8. Script Console Not Functional

- **Stated Goal**: TUI Script Console view allows loading and running Tengo scripts (USER_GUIDE.md lines 414-422)
- **Current State**: `internal/tui/app.go:282-294` renders static text showing commands (`:load`, `:list`, `:stop`) but the view doesn't accept input. No integration with `internal/script/engine.go`. Cursor shows "> _" but typing does nothing
- **Impact**: Script automation via TUI is not possible. Users must use scripts programmatically
- **Closing the Gap**:
  1. Add text input component (bubbles/textinput) to Script Console view
  2. Implement `:load <file>` command to call `engine.LoadFile()`
  3. Implement `:list` to show files in scripts directory
  4. Implement `:stop` to call `engine.Stop()`
  5. Show script output in scrollable area
  6. Add `TengoEngine` field to Model and wire up

  **Validation**:
  ```bash
  sudo ./tuimap
  # Press '4' for Script Console
  # Type ':list' - should show available scripts
  # Type ':load example.tengo' - should execute script
  ```

---

## 9. Tool View Not Interactive

- **Stated Goal**: TUI Tools View allows selecting and running network tools (USER_GUIDE.md lines 381-391)
- **Current State**: `internal/tui/app.go:267-280` renders static list of tools with message "Press number to select tool" but key handlers for 1-5 are not implemented. Selecting a tool and entering arguments is not possible
- **Impact**: Network tools cannot be used interactively from TUI. Users must use tools programmatically
- **Closing the Gap**:
  1. Add tool selection state to Model
  2. Implement key handlers for tool selection (1-5)
  3. Add text input for tool arguments
  4. Integrate with tools package Execute methods
  5. Show tool output in scrollable area

  **Validation**:
  ```bash
  sudo ./tuimap
  # Press '3' for Tools View
  # Press '1' to select netcat
  # Enter 'localhost 80' - should show connection result
  ```

---

## Priority Order

1. **TUI Scan Integration** (Gap #4) — Core functionality, directly impacts main feature claim
2. **CLI Scan Command** (Gap #6) — Essential for headless/scripted use
3. **Test Coverage** (Gap #1) — Quality gate, should be addressed alongside features
4. **Storage Integration** (Gap #5) — Enables persistence, improves usability
5. **Tool View** (Gap #9) — Completes TUI interactivity
6. **Script Console** (Gap #8) — Completes TUI interactivity
7. **Multi-Subnet** (Gap #7) — Power user feature
8. **Scan Benchmarks** (Gap #3) — Validates marketing claim
9. **NAT Port Mapping** (Gap #2) — Advanced feature, can be documented as unsupported

---

*Assessment generated 2026-04-07*
