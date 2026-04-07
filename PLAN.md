# Implementation Plan: Achieve 80% Test Coverage and Complete TUI Integration

## Project Context
- **What it does**: TuiMap is a terminal-based network diagnostic and mapping tool built in Go, designed for real-time network analysis with emphasis on speed and accuracy in NAT environments.
- **Current goal**: Achieve 80% test coverage and complete TUI-scanner integration to deliver a fully functional network discovery tool
- **Estimated Scope**: Large (28 functions above complexity 9.0, overall coverage at 44.9%, 9 documented integration gaps)

## Goal-Achievement Status
| Stated Goal | Current Status | This Plan Addresses |
|-------------|---------------|---------------------|
| Fast Network Scanning (<10s) | ✅ Implemented | Yes (benchmark validation) |
| Real-time Device Tracking | ✅ Implemented | Yes (TUI integration) |
| Integrated Network Tools | ✅ Implemented | Yes (TUI integration) |
| Extensible Scripting | ✅ Implemented | Yes (TUI integration) |
| Modern TUI Interface | ⚠️ Partial | Yes (scanner/tools integration) |
| NAT Environment Support | ✅ Implemented | No |
| >80% Test Coverage | ❌ 44.9% | Yes (primary focus) |
| CLI Scan Command | ❌ Missing | Yes |

## Metrics Summary
- **Complexity hotspots on goal-critical paths**: 28 functions above threshold (complexity >9.0)
  - `scanner.Scan` (ARP): 19.2 complexity, 76 lines
  - `scanner.pingWorker`: 14.5 complexity
  - `scanner.mergeDevices`: 14.0 complexity
  - `scanner.pingPrivileged`: 14.0 complexity
  - `tools.Execute` (dig): 12.7 complexity, 87 lines
- **Duplication ratio**: 1.03% (excellent — 53 duplicated lines total in 3 clone pairs)
- **Doc coverage**: 64.3% overall (functions: 92.9%, types: 62.0%, methods: 57.0%)
- **Package test coverage**:
  | Package | Coverage | Gap to 80% |
  |---------|----------|------------|
  | `internal/tui` | 91.4% | ✅ Met |
  | `internal/tracker` | 75.0% | 5% |
  | `internal/nat` | 67.3% | 12.7% |
  | `internal/script` | 54.5% | 25.5% |
  | `internal/config` | 48.8% | 31.2% |
  | `internal/scanner` | 37.9% | 42.1% |
  | `internal/tools` | 14.8% | 65.2% |

## Implementation Steps

### Step 1: Integrate Scanner with TUI
- **Deliverable**: Wire `scanner.Orchestrator` into TUI so pressing 's' triggers actual network scan
- **Dependencies**: None
- **Goal Impact**: Completes "Modern TUI Interface" goal — currently TUI shows "Scanning..." but no scan executes
- **Files to modify**:
  - `internal/tui/app.go`: Add `Orchestrator` field to Model, implement scan command, handle results
  - `cmd/tuimap/main.go`: Create and inject Orchestrator instance
- **Acceptance**: Pressing 's' in TUI discovers devices on local subnet, devices appear in Device List view
- **Validation**: Manual test: `sudo ./tuimap`, press 's', verify devices appear
- **Status**: ✅ COMPLETE (already implemented)

### Step 2: Add CLI Scan Command
- **Deliverable**: Implement `tuimap scan` subcommand for headless operation
- **Dependencies**: Step 1 (scanner integration patterns)
- **Goal Impact**: Enables scripted/automated scanning, fulfills documented CLI behavior
- **Files to create/modify**:
  - `cmd/tuimap/main.go`: Add `scanCmd` with flags `--subnet`, `--interface`, `--output`, `--timeout`
- **Acceptance**: `sudo tuimap scan --subnet 192.168.1.0/24 --output json` outputs discovered devices as JSON
- **Validation**: 
  ```bash
  sudo ./tuimap scan --subnet 192.168.1.0/24 --output json | jq '.[] | .ip'
  # Should list discovered IP addresses
  ```
- **Status**: ✅ COMPLETE (already implemented)

### Step 3: Add Tests for internal/tools Package (14.8% → 80%)
- **Deliverable**: Comprehensive test suite for network tools (netcat, telnet, traceroute, dig, whois)
- **Dependencies**: None
- **Goal Impact**: Addresses largest test coverage gap; tools are core functionality
- **Files to create/modify**:
  - `internal/tools/netcat_test.go`: Test TCP/UDP connections, validation, error paths
  - `internal/tools/telnet_test.go`: Test protocol negotiation, timeout handling
  - `internal/tools/traceroute_test.go`: Test hop discovery, ICMP handling
  - `internal/tools/dig_test.go`: Test DNS query types (A, AAAA, MX, TXT, NS, CNAME, PTR)
  - `internal/tools/whois_test.go`: Test domain/IP lookup, response parsing
- **Acceptance**: `go test ./internal/tools/... -cover` shows ≥80%
- **Validation**:
  ```bash
  go test -coverprofile=/tmp/tools.out ./internal/tools/...
  go tool cover -func=/tmp/tools.out | grep total
  # Must show ≥80%
  ```
- **Status**: ✅ COMPLETE (coverage at 81.5%)

### Step 4: Add Tests for internal/scanner Package (37.9% → 80%)
- **Deliverable**: Test suite covering ARP, ICMP, TCP scanners and orchestrator
- **Dependencies**: None
- **Goal Impact**: Scanner is core differentiator; tests ensure <10s scan reliability
- **Files to create/modify**:
  - `internal/scanner/arp_test.go`: Test packet construction, response parsing, worker pool
  - `internal/scanner/icmp_test.go`: Test privileged/unprivileged modes, ping worker
  - `internal/scanner/tcp_test.go`: Test port scanning, connection handling
  - `internal/scanner/orchestrator_test.go`: Test parallel execution, result merging, timeout
  - `internal/scanner/multisubnet_test.go`: Test subnet discovery, routing table parsing
- **Acceptance**: `go test ./internal/scanner/... -cover` shows ≥80%
- **Validation**:
  ```bash
  go test -coverprofile=/tmp/scanner.out ./internal/scanner/...
  go tool cover -func=/tmp/scanner.out | grep total
  # Must show ≥80%
  ```
- **Status**: ⚠️ PARTIAL (coverage at 58.5% - ARP/ICMP ping functions require root privileges for full coverage)

### Step 5: Add Tests for internal/config Package (48.8% → 80%)
- **Deliverable**: Test edge cases in configuration loading
- **Dependencies**: None
- **Goal Impact**: Configuration reliability affects all features
- **Files to modify**:
  - `internal/config/config_test.go`: Add tests for missing file, malformed YAML, env var overrides, default values
- **Acceptance**: `go test ./internal/config/... -cover` shows ≥80%
- **Validation**:
  ```bash
  go test -coverprofile=/tmp/config.out ./internal/config/...
  go tool cover -func=/tmp/config.out | grep total
  # Must show ≥80%
  ```
- **Status**: ✅ COMPLETE (80.5% coverage)

### Step 6: Add Tests for internal/script Package (54.5% → 80%)
- **Deliverable**: Test scripting engine edge cases and API functions
- **Dependencies**: None
- **Goal Impact**: Script reliability for automation use cases
- **Files to modify**:
  - `internal/script/script_test.go`: Add tests for timeout behavior, memory limits, invalid scripts, all API functions (scan, ping, portScan, resolve, alert, getDevices, get/set)
- **Acceptance**: `go test ./internal/script/... -cover` shows ≥80%
- **Validation**:
  ```bash
  go test -coverprofile=/tmp/script.out ./internal/script/...
  go tool cover -func=/tmp/script.out | grep total
  # Must show ≥80%
  ```
- **Status**: ✅ COMPLETE (81.8% coverage)

### Step 7: Add Tests for internal/nat Package (67.3% → 80%)
- **Deliverable**: Test STUN client and discovery functions
- **Dependencies**: None
- **Goal Impact**: NAT detection reliability
- **Files to modify**:
  - `internal/nat/nat_test.go`: Add tests for STUN request/response, UPnP discovery, NAT-PMP detection, error paths
- **Acceptance**: `go test ./internal/nat/... -cover` shows ≥80%
- **Validation**:
  ```bash
  go test -coverprofile=/tmp/nat.out ./internal/nat/...
  go tool cover -func=/tmp/nat.out | grep total
  # Must show ≥80%
  ```
- **Status**: ✅ COMPLETE (85.1% coverage)

### Step 8: Create Benchmark Tests for Scan Performance
- **Deliverable**: Benchmark suite validating <10s scan time claim
- **Dependencies**: Steps 1-2 (scanner integration complete)
- **Goal Impact**: Validates core marketing claim; prevents performance regressions
- **Files to create**:
  - `internal/scanner/benchmark_test.go`:
    - `BenchmarkARPScan` — target <3s for /24
    - `BenchmarkICMPScan` — target <4s for /24
    - `BenchmarkTCPScan` — target <3s for /24 with 5 ports
    - `BenchmarkOrchestrator` — target <10s end-to-end
- **Acceptance**: `go test -bench=. ./internal/scanner/...` shows orchestrator completes in <10s
- **Validation**:
  ```bash
  go test -bench=BenchmarkOrchestrator ./internal/scanner/... -benchtime=3x
  # Average time must be <10s
  ```
- **Status**: ✅ COMPLETE (benchmark_test.go created, all benchmarks pass)

### Step 9: Remove Code Duplication in ICMP Scanner
- **Deliverable**: Extract shared ICMP packet handling to reduce 28-line duplication
- **Dependencies**: Step 4 (scanner tests provide safety net)
- **Goal Impact**: Reduces maintenance burden; improves code quality
- **Files to modify**:
  - `internal/scanner/icmp.go`: Extract common logic from `pingPrivileged` (lines 136-163) and `pingUnprivileged` (lines 202-229) into helper functions
- **Acceptance**: Duplication in icmp.go reduced from 28 lines to 0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication 2>/dev/null | jq '.duplication.clones[] | select(.instances[0].file == "internal/scanner/icmp.go")'
  # Should return empty or show significantly reduced clone
  ```
- **Status**: ✅ COMPLETE (duplication reduced to 0)
- **Status**: ✅ COMPLETE (no duplication found - pingWithConn helper already extracted)

### Step 10: Integrate Storage Layer
- **Deliverable**: Wire bbolt storage for device persistence across sessions
- **Dependencies**: Step 1 (TUI integration)
- **Goal Impact**: Enables device history tracking feature
- **Files to modify**:
  - `cmd/tuimap/main.go`: Create Storage instance, pass to Registry
  - `internal/tui/app.go`: Load devices on startup, save after scans
- **Acceptance**: Devices persist between tuimap sessions
- **Validation**:
  ```bash
  # Run scan, quit, restart - previous devices should appear
  sudo ./tuimap  # press 's' to scan, 'q' to quit
  sudo ./tuimap  # devices from previous session should be visible
  ls -la ~/.local/share/tuimap/tuimap.db  # database file exists
  ```

### Step 11: Make Tool View Interactive
- **Deliverable**: Enable tool selection and execution in TUI
- **Dependencies**: Step 1 (TUI patterns established)
- **Goal Impact**: Completes "Integrated Network Tools" TUI integration
- **Files to modify**:
  - `internal/tui/app.go`: Add tool selection state, key handlers (1-5), text input for arguments, output display
- **Acceptance**: Can select and run network tools from TUI Tool View
- **Validation**: Manual test: press '3' for Tools View, '1' for netcat, enter target, see results

### Step 12: Make Script Console Interactive
- **Deliverable**: Enable script loading and execution in TUI
- **Dependencies**: Step 1 (TUI patterns), Step 6 (script tests)
- **Goal Impact**: Completes "Extensible Scripting" TUI integration
- **Files to modify**:
  - `internal/tui/app.go`: Add text input to Script Console, implement `:load`, `:list`, `:stop` commands, wire to TengoEngine
- **Acceptance**: Can load and run scripts from TUI Script Console
- **Validation**: Manual test: press '4' for Script Console, type `:list`, see available scripts

### Step 13: Migrate from google/gopacket to gopacket/gopacket
- **Deliverable**: Update to actively maintained gopacket fork
- **Dependencies**: Step 4 (scanner tests ensure compatibility)
- **Goal Impact**: Future-proofs dependency; ensures ongoing security updates
- **Files to modify**:
  - `go.mod`: Change `github.com/google/gopacket` → `github.com/gopacket/gopacket`
  - `internal/scanner/arp.go`: Update import path
- **Acceptance**: Build succeeds, all tests pass
- **Validation**:
  ```bash
  go build ./... && go test -race ./...
  # Must complete without errors
  ```

### Step 14: Improve Documentation Coverage (64.3% → 80%)
- **Deliverable**: Add godoc comments to exported types and complex functions
- **Dependencies**: None
- **Goal Impact**: Improves API discoverability for library users
- **Files to modify**:
  - `pkg/api/api.go`: Add field-level documentation to all 15 structs
  - `internal/scanner/arp.go`: Document `Scan` function (complexity 19.2)
  - `internal/scanner/orchestrator.go`: Document `mergeDevices`, `Scan`
- **Acceptance**: Documentation coverage ≥80%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation 2>/dev/null | jq '.documentation.coverage.overall'
  # Must show ≥80
  ```

## Dependency Graph

```
Step 1 (TUI-Scanner) ─────────────────┬─> Step 2 (CLI Scan)
                                      ├─> Step 8 (Benchmarks)
                                      ├─> Step 10 (Storage)
                                      ├─> Step 11 (Tool View)
                                      └─> Step 12 (Script Console)

Step 3 (tools tests) ─────────────────┬─> [independent]
Step 4 (scanner tests) ───────────────┼─> Step 9 (ICMP refactor)
                                      └─> Step 13 (gopacket migration)
Step 5 (config tests) ────────────────┬─> [independent]
Step 6 (script tests) ────────────────┼─> Step 12 (Script Console)
Step 7 (nat tests) ───────────────────┴─> [independent]

Step 14 (documentation) ──────────────── [independent]
```

## Estimated Effort

| Step | Description | Estimated Duration | Priority |
|------|-------------|-------------------|----------|
| 1 | TUI-Scanner Integration | 1 day | P0 |
| 2 | CLI Scan Command | 0.5 days | P0 |
| 3 | Tools Tests | 2 days | P1 |
| 4 | Scanner Tests | 3 days | P1 |
| 5 | Config Tests | 0.5 days | P1 |
| 6 | Script Tests | 1 day | P1 |
| 7 | NAT Tests | 0.5 days | P1 |
| 8 | Benchmark Tests | 0.5 days | P2 |
| 9 | ICMP Refactor | 0.5 days | P2 |
| 10 | Storage Integration | 0.5 days | P2 |
| 11 | Tool View Interactive | 1 day | P2 |
| 12 | Script Console Interactive | 1 day | P2 |
| 13 | gopacket Migration | 0.5 days | P3 |
| 14 | Documentation | 1 day | P3 |
| **Total** | | **~14 days** | |

## Risk Mitigation

| Risk | Mitigation Strategy |
|------|---------------------|
| Test coverage effort exceeds estimate | Prioritize tools package (largest gap); accept 70% as interim target if needed |
| Scanner integration breaks TUI | Add integration tests; implement feature flags for gradual rollout |
| Benchmark shows >10s scan time | Profile with pprof; optimize hot paths; implement early-exit when 99% confident |
| gopacket migration breaks ARP scanner | Run migration after scanner tests achieve 80%; pin to specific version |
| ICMP refactor introduces regressions | Only refactor after scanner tests provide safety net |

## Success Criteria

1. **Test coverage**: `go test -coverprofile=coverage.out ./...` shows ≥80% total
2. **Functionality**: TUI scan discovers devices, CLI scan outputs JSON
3. **Performance**: Benchmark validates <10s scan for /24 subnet
4. **Quality**: No increase in complexity hotspots, duplication ratio stays <2%

---

*Plan generated 2026-04-07 using go-stats-generator v1.0.0 metrics and project goal analysis*
