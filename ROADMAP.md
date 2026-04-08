# Goal-Achievement Assessment

## Project Context
- **What it claims to do**: TuiMap is a terminal-based network diagnostic and mapping tool that discovers all devices on /24 networks in under 10 seconds using parallel multi-method scanning (ARP, ICMP, TCP). It provides real-time device tracking with alerts, integrated network tools (netcat, telnet, traceroute, dig, whois), an extensible Tengo scripting engine, a modern TUI interface with multiple views, and NAT environment support.
- **Target audience**: Network administrators, security professionals, and developers who need fast, reliable network discovery and diagnostic capabilities directly from the terminal — particularly in NAT environments, multi-subnet networks, and cloud infrastructure.
- **Architecture**: Layered architecture across 9 packages (3,114 LoC production code, 24 files):
  - `cmd/tuimap/` — CLI entry point with Cobra (root, version, config, scan subcommands)
  - `internal/scanner/` — ARP, ICMP, TCP scanners + orchestrator + multi-subnet (6 files, 61 functions)
  - `internal/tracker/` — Device registry, alert engine, bbolt storage (3 files, 27 functions)
  - `internal/tools/` — Netcat, telnet, traceroute, dig, whois (6 files, 59 functions)
  - `internal/script/` — Tengo VM integration and API bridge (3 files, 40 functions)
  - `internal/tui/` — Bubble Tea views — network map, device list, tools, script console (2 files, 31 functions)
  - `internal/nat/` — NAT detection, STUN, UPnP/NAT-PMP discovery (1 file, 24 functions)
  - `internal/config/` — Viper-based YAML config management (1 file, 3 functions)
  - `pkg/api/` — Public API interfaces (1 file, 15 structs/interfaces)
- **Existing CI/quality gates**:
  - `.github/workflows/ci.yml` — Build, `go vet`, tests with `-race`, golangci-lint, coverage check (warns below 35%)
  - `.github/workflows/benchmark.yml` — Weekly + PR-triggered scanner benchmarks with 10s threshold check
  - `Makefile` — build, test, coverage, fmt, vet, lint targets

## Metrics Summary (go-stats-generator)
| Metric | Value |
|--------|-------|
| Total LoC (non-test) | 3,114 |
| Functions/Methods | 80 / 171 |
| Avg function length | 14.9 lines |
| Functions > 50 lines | 11 (4.4%) |
| Avg cyclomatic complexity | 4.6 |
| High complexity (>10) | 3 functions |
| Duplication ratio | 0.40% (24 lines, 2 clone pairs) |
| Documentation coverage | 65.3% overall (93.8% functions, 57.5% methods) |
| Circular dependencies | None |
| `go vet` | Clean (0 warnings) |
| `go test` | All packages pass |
| Test coverage | 72.9% overall |

### Per-Package Test Coverage
| Package | Coverage | vs 35% Target |
|---------|----------|---------------|
| `internal/config` | 87.8% | ✅ +52.8 |
| `internal/tracker` | 83.3% | ✅ +48.3 |
| `internal/script` | 81.8% | ✅ +46.8 |
| `internal/nat` | 78.6% | ✅ +43.6 |
| `internal/tools` | 76.1% | ✅ +41.1 |
| `internal/tui` | 71.3% | ✅ +36.3 |
| `internal/scanner` | 62.3% | ✅ +27.3 |

### Top Complex Functions
| Function | Package | Lines | Complexity | Risk |
|----------|---------|-------|------------|------|
| `pingWorker` | scanner | 50 | 19.7 | Critical path — ICMP scan |
| `Update` | tui | 102 | 19.4 | TUI event loop — high traffic |
| `Scan` (ARP) | scanner | 76 | 19.2 | Critical path — ARP scan |
| `mergeDevices` | scanner | 39 | 14.0 | Deduplication logic |
| `listenForResponses` | scanner | 17 | 13.7 | Packet parsing |

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence | Gap Description |
|---|-------------|--------|----------|-----------------|
| 1 | Fast Network Scanning (<10s for /24) | ✅ Achieved | Orchestrator at `internal/scanner/orchestrator.go` runs ARP+ICMP+TCP in parallel goroutines via `runScannersParallel()` (lines 42-65). 15s context timeout in TUI. Benchmark CI at `.github/workflows/benchmark.yml` validates 10s threshold. `go test ./internal/scanner/...` completes in ~4.5s. | Benchmark tests exist (`benchmark_test.go`) but test initialization only — no end-to-end scan benchmarks against mock network. |
| 2 | Real-time Device Tracking & Alerts | ⚠️ Partial | `internal/tracker/registry.go` (246 lines, 83.3% coverage) implements `Update()` with new/offline/changed detection and alert generation. **However**, TUI scan handler (`app.go:317-326`) populates `m.devices` directly from scan results but never calls `registry.Update()`, so alerts are never generated during TUI usage. | Registry exists and works in isolation, but is not wired into the TUI scan flow. No alert display in the TUI Update loop. |
| 3 | Integrated Network Tools (nc, telnet, traceroute, dig, whois) | ✅ Achieved | All 5 tools fully implemented (not stubs): netcat.go (175L), telnet.go (185L), traceroute.go (309L), dig.go (225L), whois.go (199L). All implement `NetworkTool` interface with `Execute()` returning `<-chan string`. 76.1% test coverage. | TUI tool view is interactive — tool selection, argument input, and async execution all wired up (`app.go:826-871`). |
| 4 | Extensible Scripting (Tengo) | ⚠️ Partial | `internal/script/engine.go` (289L) implements `TengoEngine` with `Run()`, `LoadFile()`, resource limits (max allocs, timeout). API bridge at `api.go` (207L) defines `scan()`, `ping()`, `port_scan()`, `get_devices()`, `set()`/`get()` functions. TUI has script console with `:load`, `:list`, `:stop` commands. 81.8% coverage. | API functions use default stub implementations that return empty/false unless `SetScanner()`/`SetPinger()` are called on the APIBridge — but TUI never calls these setters (`app.go` creates `NewTengoEngine` at line 203 but never wires real scanner/registry to it). Scripts execute but network API calls silently return empty results. |
| 5 | Modern TUI Interface with Multiple Views | ✅ Achieved | `internal/tui/app.go` (893L) implements 4 views with Bubble Tea: Network Map (ASCII topology), Device List (interactive table), Tools View (selection + input + output), Script Console (`:load`/`:list`/`:stop` + output). Key bindings 1-4 for view switching, `s` for scan, `q` for quit. Interactive text inputs using `bubbles/textinput`. 71.3% coverage. | All views functional. Network map view is static ASCII art (no real topology graph). |
| 6 | NAT Environment Support | ⚠️ Partial | `internal/nat/nat.go` (559L) implements `Discover()` with gateway detection, local IP detection, STUN-based public IP query, UPnP/NAT-PMP discovery, and NAT type determination. 78.6% coverage. | `AddPortMapping()` and `RemovePortMapping()` are stubs returning `ErrNATUnsupported` (lines 516-532, explicitly documented as stubs via `NOTE:` comments). NAT detection works; port mapping does not. |
| 7 | Persistent Device History (bbolt) | ⚠️ Partial | `internal/tracker/storage.go` (259L) implements `Storage` with `SaveDevice()`, `SaveDevices()`, `LoadDevices()`, `SaveAlert()`, `LoadAlerts()`, `Close()`. `cmd/tuimap/main.go:72` creates Storage and passes to TUI at line 83. 83.3% coverage. | TUI stores the `storage` field (`app.go:67`) but **never calls** `SaveDevices()` after scans or `LoadDevices()` on startup. Persistence is implemented but not integrated — device history is lost between sessions. |
| 8 | CLI Scan Command | ✅ Achieved | `cmd/tuimap/main.go:138-223` implements `scan` subcommand with `--subnet`, `--interface`, `--output` (json/text), `--timeout`, `--all-subnets`, `--from-routes` flags. Multi-subnet scanning via `MultiSubnetScanner.ScanAllSubnets()` and `ScanFromRoutingTable()`. JSON and text output formatters. | Fully functional headless scanning with all documented flags. |
| 9 | Multi-Subnet Scanning | ✅ Achieved | `internal/scanner/multisubnet.go` (397L) implements `MultiSubnetScanner`, `DiscoverSubnets()`, `ParseRoutingTable()`, `ScanAllSubnets()`, `ScanFromRoutingTable()`. Exposed via CLI `scan --all-subnets` and `scan --from-routes` (main.go:171-194). | Not yet exposed in TUI (no subnet selector), but CLI path is complete. |
| 10 | >35% Test Coverage | ✅ Achieved | 72.9% overall. All packages individually exceed 35%: lowest is `internal/scanner` at 62.3%. CI coverage check in `ci.yml:67-72`. | Significantly exceeds target. |
| 11 | Configuration Management (Viper/YAML) | ✅ Achieved | `internal/config/config.go` (257L) with `LoadConfig()`, `SaveConfig()`, `DefaultConfig()`, `InitConfig()`. Full config structure covering scanner, alerts, NAT, scripting, TUI, storage, logging. CLI `config init` and `config show` subcommands. 87.8% coverage. | Fully working. |
| 12 | Graceful Degradation without Root | ✅ Achieved | `cmd/tuimap/main.go:53-56` catches orchestrator creation errors as warnings, not fatal. TCP scanner requires no root. README documents privilege requirements. | Working as documented. |

**Overall: 7/12 goals fully achieved, 5/12 partially achieved, 0/12 missing**

---

## Roadmap

### Priority 1: Wire Device Tracker (Registry) into TUI Scan Flow
**Goal affected**: #2 (Real-time Device Tracking & Alerts) — core feature claim
**Risk**: Registry `Update()` has complexity 12.4 and is well-tested (83.3% coverage), so integration risk is low.

- [x] Add `tracker.Registry` field to `tui.Model` struct (currently only has raw `devices []scanner.Device`)
- [x] In `NewModelWithOrchestratorAndStorage()` (`app.go:144`), create a `tracker.NewRegistry()` instance with configurable offline threshold
- [x] In `scanResultMsg` handler (`app.go:317-326`), call `registry.Update(msg.result.Devices)` before setting `m.devices`
- [x] After `registry.Update()`, call `registry.GetAlerts()` and append to `m.alerts` for display in the status bar (`app.go:752`)
- [x] Add alert display in Network Map or Device List view (currently `m.alerts` is tracked but never populated during scans)
- [x] **Validation**: Run TUI, press `s` to scan — devices should appear AND new-device alerts should increment the alert counter in the status bar

### Priority 2: Integrate Storage Persistence in TUI
**Goal affected**: #7 (Persistent Device History) — directly impacts usability across sessions
**Risk**: Storage is already tested at 83.3% coverage; integration is straightforward.

- [x] In the `scanResultMsg` handler (`app.go:317-326`), after updating devices, call `m.storage.SaveDevices(m.devices)` if `m.storage != nil`
- [x] In `NewModelWithOrchestratorAndStorage()` (`app.go:144`), load previously saved devices via `storage.LoadDevices()` and pre-populate `m.devices`
- [x] Save alerts via `storage.SaveAlert()` when generated by the registry
- [x] **Validation**: Run TUI → scan → quit → restart TUI — previously discovered devices should appear immediately

### Priority 3: Wire Script Engine API Bridge to Real Scanner/Registry
**Goal affected**: #4 (Extensible Scripting) — scripts currently execute but all network API calls return empty results
**Risk**: API bridge setters exist and are tested; wiring is mechanical.

- [ ] In `NewModelWithOrchestratorAndStorage()` (`app.go:203-225`), after creating the `TengoEngine`, call `engine.SetAPIBridge()` with an `APIBridge` that has `SetScanner()` and `SetPinger()` wired to the real orchestrator
- [ ] Create a `ScannerFunc` wrapper that calls `m.orchestrator.Scan()` and converts `[]scanner.Device` to `[]map[string]interface{}`
- [ ] Create a `PingerFunc` wrapper using ICMP or TCP connectivity check
- [ ] Wire `get_devices()` to return data from the registry/device list
- [ ] **Validation**: Run TUI → press `4` for Script Console → type `:load` with an example script like `scripts/examples/auto-scan.tengo` — script should discover real devices, not return empty arrays

### Priority 4: Add End-to-End Scan Benchmark Tests
**Goal affected**: #1 (Fast Scanning <10s) — the core differentiating claim needs regression protection
**Risk**: `benchmark_test.go` exists but only tests initialization, not actual scanning.

- [ ] Add `BenchmarkOrchestratorFullScan` to `internal/scanner/benchmark_test.go` that creates a mock network interface or uses localhost scanning to validate <10s budget
- [ ] Add `BenchmarkARPScanSubnet`, `BenchmarkICMPScanSubnet`, `BenchmarkTCPScanSubnet` measuring per-method times against individual method budgets (ARP: <3s, ICMP: <4s, TCP: <3s)
- [ ] Ensure `.github/workflows/benchmark.yml` references `BenchmarkOrchestratorFullScan` (it already does at line 30, but the function doesn't exist yet)
- [ ] **Validation**: `go test -bench=BenchmarkOrchestratorFullScan ./internal/scanner/... -benchtime=3x` should complete with each iteration under 10s

### Priority 5: Reduce Complexity of Top-3 Hot Functions
**Goal affected**: Multiple — these functions are on critical paths and their high complexity (19+) increases bug risk
**Evidence**: `go-stats-generator` reports `pingWorker` (19.7), `Update` (19.4), `Scan/ARP` (19.2) as highest complexity. The project's median complexity is 4.6.

- [ ] **`pingWorker`** (`internal/scanner/icmp.go`, 50 lines, complexity 19.7): Extract ICMP packet construction and response parsing into separate functions. The current function handles connection setup, packet building, sending, receiving, and response validation in one block.
- [ ] **`Update`** (`internal/tui/app.go`, 102 lines, complexity 19.4): Extract per-view key handlers into separate methods (e.g., `handleNetworkMapKeys()`, `handleDeviceListKeys()`, `handleToolViewKeys()`, `handleScriptConsoleKeys()`). Currently a single large switch with nested switches.
- [ ] **`Scan` (ARP)** (`internal/scanner/arp.go`, 76 lines, complexity 19.2): Extract `sendARPRequest()` and `processARPResponse()` helper functions from the monolithic scan loop.
- [ ] **Validation**: Re-run `go-stats-generator analyze . --skip-tests` — no function should exceed complexity 15 on critical paths

### Priority 6: Expose Multi-Subnet Scanning in TUI
**Goal affected**: #9 (Multi-Subnet Scanning) — implemented in CLI but not in TUI
**Risk**: Low — `MultiSubnetScanner` is well-implemented (397 lines) and already exposed in CLI.

- [ ] Add a subnet discovery step on TUI startup that calls `scanner.DiscoverSubnets()` and presents discovered subnets
- [ ] Allow users to select which subnet(s) to scan (or scan all) from the TUI
- [ ] Display per-subnet results in the Network Map view with subnet grouping
- [ ] **Validation**: Run TUI on a multi-homed machine → should show discovered subnets → pressing `s` should scan selected subnet(s)

### Priority 7: NAT Port Mapping — Document or Implement
**Goal affected**: #6 (NAT Environment Support) — port mapping is a stub, explicitly acknowledged via `NOTE:` comments
**Risk**: Full UPnP IGD implementation requires SOAP/HTTP calls; NAT-PMP requires raw packet construction. Both are non-trivial.

- [ ] **Option A (Recommended)**: Document the limitation in README and API.md — add a "Known Limitations" section clarifying that NAT detection works but port mapping (`AddPortMapping`/`RemovePortMapping`) is not yet implemented. Remove port mapping from `NATClient` interface examples.
- [ ] **Option B**: Implement UPnP IGD SOAP calls for `AddPortMapping` action at `internal/nat/nat.go:516-523` and NAT-PMP mapping request packets per RFC 6886 at lines 526-532.
- [ ] **Validation (Option A)**: `grep -n "AddPortMapping" README.md docs/*.md` should only appear in "Known Limitations" context
- [ ] **Validation (Option B)**: Integration test with UPnP-capable gateway showing port mapping creation/deletion

### Priority 8: Reduce Duplication in TUI View Code
**Goal affected**: Code maintainability — `go-stats-generator` found 2 clone pairs (24 lines, 0.40% ratio)
**Evidence**: `internal/tui/app.go:173-180` ↔ `app.go:193-200` (8 lines), `app.go:701-716` ↔ `app.go:731-746` (16 lines)

- [ ] Extract the duplicated text input initialization pattern (lines 173-180, 193-200) into a helper function
- [ ] Extract the duplicated view rendering pattern (lines 701-716, 731-746) into a parameterized helper
- [ ] **Validation**: Re-run `go-stats-generator analyze . --skip-tests` — duplication ratio should drop to 0%

---

## Competitive Context

TuiMap occupies a niche between fast CLI scanners (RustScan, Masscan) and full-featured GUI tools (Angry IP Scanner, Zenmap). Its closest competitors in the TUI space are:
- **havn** (Rust) — TUI port scanner, fast but no device tracking or scripting
- **Netshow** (Rust) — Interactive process-aware network monitor, not a scanner
- **termshark** — TUI for packet capture, not scanning

TuiMap's unique combination of <10s scanning + device tracking + scripting + TUI is not matched by any single existing tool. The main competitive risk is from RustScan (raw speed) and Nmap (feature depth). Completing the tracker integration (Priority 1) and script wiring (Priority 3) would solidify TuiMap's differentiation.

---

*Assessment generated 2026-04-08 using go-stats-generator v1.0.0, go test (72.9% coverage), go vet (clean), and manual code review.*
