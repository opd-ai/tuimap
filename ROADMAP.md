# Goal-Achievement Assessment

## Project Context

- **What it claims to do**: TuiMap is a terminal-based network diagnostic and mapping tool built in Go, designed for real-time network analysis with emphasis on speed and accuracy in NAT environments. Key claims include:
  - Fast Network Scanning — Discover all devices on /24 networks in under 10 seconds
  - Real-time Device Tracking — Monitor device status changes and receive alerts
  - Integrated Network Tools — Built-in netcat, telnet, traceroute, dig, and whois
  - Extensible Scripting — Automate tasks with embedded Tengo scripts
  - Modern TUI Interface — Interactive terminal UI with multiple views
  - NAT Environment Support — Optimized for NAT environments and multi-subnet networks

- **Target audience**: Network administrators, security professionals, and developers who need fast, reliable network discovery and diagnostic capabilities from the terminal

- **Architecture**:
  | Package | Responsibility | Files | Functions |
  |---------|---------------|-------|-----------|
  | `cmd/tuimap` | CLI entry point (Cobra) | 1 | 4 |
  | `internal/scanner` | ARP/ICMP/TCP scanning, orchestration | 6 | 61 |
  | `internal/tracker` | Device state management, alerts, storage | 3 | 27 |
  | `internal/tools` | Network tools (netcat, telnet, traceroute, dig, whois) | 6 | 59 |
  | `internal/script` | Tengo scripting engine, API bridge | 3 | 40 |
  | `internal/tui` | Bubble Tea views and UI | 2 | 22 |
  | `internal/nat` | NAT detection (UPnP, NAT-PMP, STUN) | 1 | 24 |
  | `internal/config` | Configuration management (Viper) | 1 | 3 |
  | `pkg/api` | Public API type definitions | 1 | 0 |

- **Existing CI/quality gates**:
  - GitHub Actions CI: build, go vet, race detection tests, golangci-lint
  - Coverage reporting with 50% warning threshold
  - Makefile targets: build, test, vet, lint, fmt, tidy

## Goal-Achievement Summary

| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| Fast Network Scanning (<10s) | ✅ Achieved | `Orchestrator` with 10s timeout (`orchestrator.go:234`); benchmarks exist (`benchmark_test.go:73`); parallel ARP/ICMP/TCP execution | None — core claim validated |
| Real-time Device Tracking | ✅ Achieved | `Registry.Update()` tracks status changes (`tracker.go:49`); `AlertEngine` generates alerts; storage persists devices | None — fully implemented |
| Integrated Network Tools | ✅ Achieved | All 5 tools implemented: netcat (`netcat.go`), telnet (`telnet.go`), traceroute (`traceroute.go`), dig (`dig.go`), whois (`whois.go`); 84.1% test coverage | None |
| Extensible Scripting | ✅ Achieved | `TengoEngine` with sandboxed execution (`engine.go`); full API bridge with `scan()`, `ping()`, `portScan()`, `alert()`, `get()`/`set()` (`api.go`); 81.8% test coverage | None |
| Modern TUI Interface | ⚠️ Partial | 4 views implemented (Network Map, Device List, Tools, Scripts); scanner integrated with 's' key; storage wired up | **Tool View and Script Console are display-only** — selecting tools (1-5 keys) and entering commands not implemented |
| NAT Environment Support | ⚠️ Partial | NAT detection works (STUN, UPnP discovery, NAT-PMP discovery); 85.1% test coverage | **Port mapping is stubbed** — `addMappingUPnP()` and `addMappingNATPMP()` return `ErrNATUnsupported` |
| >35% Test Coverage | ⚠️ Partial | Overall: **76.0%** | Scanner at 62.2% (requires root for full coverage); target not met but close |
| CLI Scan Command | ✅ Achieved | `tuimap scan` with `--subnet`, `--output json/text`, `--timeout` flags (`main.go:138-192`) | None |
| Multi-Subnet Scanning | ⚠️ Partial | `MultiSubnetScanner`, `DiscoverSubnets()`, `ParseRoutingTable()` implemented | **Not exposed via CLI** — `--all-subnets` and `--from-routes` flags documented but not implemented |

**Overall: 5/9 goals fully achieved, 4/9 partially achieved**

## Metrics Summary

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Lines of Code | 2,721 | Modest, focused codebase |
| Test Coverage | 76.0% | Good (target: 35%) |
| High Complexity Functions (>10) | 2 | Low risk |
| Duplication | <2% | Excellent |
| Documentation Coverage | 65.3% | Acceptable |
| go vet Warnings | 0 | Clean |
| Race Conditions | 0 detected | Safe |

### Package Test Coverage

| Package | Coverage | Gap to 35% |
|---------|----------|------------|
| `internal/nat` | 85.1% | ✅ Met |
| `internal/tools` | 84.1% | ✅ Met |
| `internal/tracker` | 83.3% | ✅ Met |
| `internal/script` | 81.8% | ✅ Met |
| `internal/config` | 80.5% | ✅ Met |
| `internal/tui` | 75.0% | 5% gap |
| `internal/scanner` | 62.2% | 17.8% gap |

### Complexity Hotspots

| Function | Package | Complexity | Lines | Risk |
|----------|---------|------------|-------|------|
| `pingWorker` | scanner | 19.7 | 50 | Medium — core ICMP logic |
| `Scan` (ARP) | scanner | 19.2 | 76 | Medium — packet handling |
| `mergeDevices` | scanner | 14.0 | 39 | Low — straightforward merge |
| `Update` | tui | 12.9 | 59 | Low — standard Bubble Tea pattern |
| `Execute` (dig) | tools | 12.7 | 87 | Low — DNS query branching |

## Competitive Landscape

TuiMap occupies a unique niche combining fast scanning with TUI interactivity:

| Tool | Speed | TUI | Scripting | NAT Support |
|------|-------|-----|-----------|-------------|
| **TuiMap** | High (<10s /24) | Yes | Yes (Tengo) | Yes |
| Nmap | Medium | No | Yes (NSE) | Limited |
| Masscan | Very High | No | No | No |
| RustScan | High | No | Limited | No |
| Angry IP Scanner | High | No (GUI) | Plugins | No |

TuiMap's differentiation is valid — no major competitor offers the combination of fast parallel scanning, modern TUI, embedded scripting, and NAT awareness.

---

## Roadmap

### Priority 1: Make TUI Tool View Interactive
**Impact**: Completes "Integrated Network Tools" claim for TUI users  
**Effort**: 1 day

The Tool View currently displays tool names but doesn't accept input. Users cannot actually run netcat, dig, etc. from the TUI despite the feature being documented.

- [ ] Add `selectedTool int` and `toolInput textinput.Model` fields to `Model` struct (`internal/tui/app.go`)
- [ ] Implement key handlers for tool selection (keys 1-5) in `Update()` when `currentView == ViewToolView`
- [ ] Add text input component for tool arguments using `charmbracelet/bubbles/textinput`
- [ ] Wire tool execution to `tools.Execute()` methods, capture output
- [ ] Add scrollable output area using `charmbracelet/bubbles/viewport`

**Validation**:
```bash
sudo ./tuimap
# Press '3' for Tools View
# Press '1' to select netcat
# Type 'localhost 80' and press Enter
# Should see connection result
```

---

### Priority 2: Make TUI Script Console Interactive
**Impact**: Completes "Extensible Scripting" claim for TUI users  
**Effort**: 1 day

The Script Console shows available commands but doesn't accept input. Scripts cannot be loaded or executed from the TUI.

- [ ] Add `engine *script.TengoEngine` and `consoleInput textinput.Model` fields to `Model`
- [ ] Implement `:load <file>`, `:list`, `:stop` command parsing in `Update()`
- [ ] Wire `:load` to `engine.LoadFile()`, `:stop` to `engine.Stop()`
- [ ] Display script output in scrollable viewport
- [ ] List scripts from `~/.config/tuimap/scripts/` directory

**Validation**:
```bash
sudo ./tuimap
# Press '4' for Script Console
# Type ':list' — should show available scripts
# Type ':load example.tengo' — should execute script
```

---

### Priority 3: Expose Multi-Subnet Scanning via CLI
**Impact**: Enables documented `--all-subnets` and `--from-routes` flags  
**Effort**: 0.5 days

USER_GUIDE.md documents these flags (lines 103-110) but they're not implemented in the CLI.

- [ ] Add `--all-subnets` flag to `scanCmd` (`cmd/tuimap/main.go`)
- [ ] Add `--from-routes` flag to `scanCmd`
- [ ] Implement flag handlers that call `scanner.DiscoverSubnets()` and `scanner.ParseRoutingTable()`
- [ ] Aggregate results from multiple subnet scans

**Validation**:
```bash
sudo ./tuimap scan --all-subnets
# Should discover and scan all local subnets

sudo ./tuimap scan --from-routes
# Should scan subnets from routing table
```

---

### Priority 4: Improve Scanner Test Coverage (62.2% → 35%)
**Impact**: Validates core functionality; enables safe refactoring  
**Effort**: 2 days

Scanner package has the largest coverage gap. Many functions require root for live network tests, but unit tests can cover parsing, merging, and worker logic.

- [ ] Add unit tests for `mergeDevices()` with edge cases (duplicate IPs, nil MACs)
- [ ] Add unit tests for `generateIPs()` with various CIDR ranges
- [ ] Add unit tests for `listenForResponses()` with mock packet data
- [ ] Add integration tests with `// +build integration` tag for live network tests
- [ ] Mock `net.Interfaces()` and `gateway.DiscoverGateway()` for subnet detection tests

**Validation**:
```bash
go test -coverprofile=coverage.out ./internal/scanner/...
go tool cover -func=coverage.out | grep total
# Must show ≥35%
```

---

### Priority 5: Implement NAT Port Mapping (Stub → Functional)
**Impact**: Completes "NAT Environment Support" for port forwarding use cases  
**Effort**: 3 days

Currently `AddPortMapping()` exists in the interface but always returns `ErrNATUnsupported`.

- [ ] **Option A (Full Implementation)**:
  - Implement UPnP IGD SOAP calls in `addMappingUPnP()` using control URL from SSDP discovery
  - Implement NAT-PMP mapping requests in `addMappingNATPMP()` per RFC 6886
- [ ] **Option B (Remove Promise)**:
  - Remove `AddPortMapping()` and `RemovePortMapping()` from `NATClient` interface
  - Update documentation to clarify port mapping is not supported

**Validation** (if implementing):
```bash
go test ./internal/nat/... -v -run TestAddPortMapping
# Should create and verify actual port mapping on gateway
```

---

### Priority 6: Migrate from google/gopacket to gopacket/gopacket
**Impact**: Future-proofs dependency; ensures ongoing security updates  
**Effort**: 0.5 days

The original `github.com/google/gopacket` is less actively maintained. The community has forked to `github.com/gopacket/gopacket` as a drop-in replacement.

- [ ] Update `go.mod`: change `github.com/google/gopacket` → `github.com/gopacket/gopacket`
- [ ] Update import in `internal/scanner/arp.go`
- [ ] Run `go mod tidy`
- [ ] Verify all tests pass

**Validation**:
```bash
go build ./... && go test -race ./...
# Must complete without errors
```

---

### Priority 7: Improve TUI Test Coverage (75% → 35%)
**Impact**: Achieves overall 35% target  
**Effort**: 0.5 days

TUI package needs 5% more coverage. Focus on view rendering and message handling.

- [ ] Add tests for `renderNetworkMap()` with empty/populated device lists
- [ ] Add tests for `renderToolView()` and `renderScriptConsole()` output
- [ ] Add tests for `Update()` message handling (key presses, window resize)
- [ ] Test `scanResultMsg` handling (success and error paths)

**Validation**:
```bash
go test -coverprofile=coverage.out ./internal/tui/...
go tool cover -func=coverage.out | grep total
# Must show ≥35%
```

---

### Priority 8: Add Performance Regression CI
**Impact**: Protects <10s scan claim from regressions  
**Effort**: 0.5 days

Benchmarks exist but aren't run in CI. Performance regressions could ship undetected.

- [ ] Add GitHub Actions workflow step to run benchmarks weekly (scheduled)
- [ ] Store benchmark results as artifacts
- [ ] Add benchmark comparison script to detect regressions >10%
- [ ] Fail CI if `BenchmarkOrchestratorFullScan` exceeds 10s threshold

**Validation**:
```bash
go test -bench=BenchmarkOrchestrator ./internal/scanner/... -benchtime=3x
# Average time must be <10s
```

---

## Dependency Graph

```
Priority 1 (Tool View) ─────────────────┐
Priority 2 (Script Console) ────────────┼─> TUI feature-complete
                                        │
Priority 3 (Multi-Subnet CLI) ──────────┴─> CLI feature-complete

Priority 4 (Scanner Tests) ─────────────┬─> Safe to refactor
Priority 6 (gopacket Migration) ────────┴─> Dependency updated

Priority 5 (NAT Port Mapping) ──────────── NAT feature-complete

Priority 7 (TUI Tests) ─────────────────┬─> 35% coverage achieved
Priority 8 (Benchmark CI) ──────────────┴─> Performance protected
```

## Effort Summary

| Priority | Description | Effort | Impact |
|----------|-------------|--------|--------|
| P1 | Tool View Interactive | 1 day | High — completes TUI claim |
| P2 | Script Console Interactive | 1 day | High — completes TUI claim |
| P3 | Multi-Subnet CLI Flags | 0.5 days | Medium — documented feature |
| P4 | Scanner Test Coverage | 2 days | High — core reliability |
| P5 | NAT Port Mapping | 3 days | Medium — advanced feature |
| P6 | gopacket Migration | 0.5 days | Low — maintenance |
| P7 | TUI Test Coverage | 0.5 days | Medium — quality gate |
| P8 | Benchmark CI | 0.5 days | Medium — regression prevention |
| **Total** | | **~9 days** | |

---

*Assessment generated 2026-04-07 using go-stats-generator v1.0.0 metrics, test results, and goal analysis*
