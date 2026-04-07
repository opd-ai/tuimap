//go:build integration

// Package scanner provides network scanning functionality for device discovery.
// Integration tests in this file require real network access.
package scanner

import (
	"context"
	"testing"
	"time"
)

// TestIntegrationTCPScan performs a real TCP scan against localhost.
func TestIntegrationTCPScan(t *testing.T) {
	scanner := NewTCPScanner(10, 500*time.Millisecond, []int{22, 80, 443})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Scan localhost - should always succeed
	devices, err := scanner.Scan(ctx, "127.0.0.1/32")
	if err != nil {
		t.Fatalf("TCP scan failed: %v", err)
	}

	// Localhost may or may not have open ports
	t.Logf("Found %d devices on localhost", len(devices))
}

// TestIntegrationICMPScan performs a real ICMP ping against localhost.
func TestIntegrationICMPScan(t *testing.T) {
	scanner := NewICMPScanner(10, 500*time.Millisecond, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Scan localhost - should always respond to ping
	devices, err := scanner.Scan(ctx, "127.0.0.1/32")
	if err != nil {
		t.Fatalf("ICMP scan failed: %v", err)
	}

	t.Logf("Found %d devices via ICMP on localhost", len(devices))
}

// TestIntegrationOrchestrator tests the full orchestrated scan.
func TestIntegrationOrchestrator(t *testing.T) {
	orch := NewOrchestrator(5 * time.Second)

	// Add scanners
	tcpScanner := NewTCPScanner(10, 500*time.Millisecond, nil)
	orch.AddScanner(tcpScanner)

	icmpScanner := NewICMPScanner(10, 500*time.Millisecond, 1)
	orch.AddScanner(icmpScanner)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Scan localhost subnet
	result, err := orch.Scan(ctx, "127.0.0.1/32")
	if err != nil {
		t.Fatalf("Orchestrated scan failed: %v", err)
	}

	t.Logf("Scan completed in %v", result.ScanTime)
	t.Logf("Found %d devices", len(result.Devices))
	t.Logf("Method: %s", result.Method)
}

// TestIntegrationDetectSubnet tests subnet detection.
func TestIntegrationDetectSubnet(t *testing.T) {
	subnet, err := DetectSubnet()
	if err != nil {
		t.Fatalf("DetectSubnet failed: %v", err)
	}

	t.Logf("Detected subnet: %s", subnet)

	// Validate it looks like a valid subnet
	if subnet == "" {
		t.Error("Detected subnet is empty")
	}
}

// TestIntegrationDiscoverSubnets tests multi-subnet discovery.
func TestIntegrationDiscoverSubnets(t *testing.T) {
	subnets, err := DiscoverSubnets()
	if err != nil {
		t.Fatalf("DiscoverSubnets failed: %v", err)
	}

	t.Logf("Found %d subnets:", len(subnets))
	for _, subnet := range subnets {
		t.Logf("  - %s on %s (local=%v)", subnet.Subnet, subnet.Interface, subnet.Local)
	}
}

// TestIntegrationFullScanTiming tests that scans complete within time budget.
func TestIntegrationFullScanTiming(t *testing.T) {
	orch, err := CreateDefaultOrchestrator("")
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()
	start := time.Now()

	// Scan a small subnet
	result, err := orch.Scan(ctx, "127.0.0.0/30")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	t.Logf("Full scan completed in %v", elapsed)
	t.Logf("Reported scan time: %v", result.ScanTime)

	// Should complete well under 10 seconds for tiny subnet
	if elapsed > 15*time.Second {
		t.Errorf("Scan took too long: %v (expected <15s)", elapsed)
	}
}

// TestIntegrationParseRoutingTable tests routing table parsing.
func TestIntegrationParseRoutingTable(t *testing.T) {
	subnets, err := ParseRoutingTable()
	if err != nil {
		t.Fatalf("ParseRoutingTable failed: %v", err)
	}

	t.Logf("Found %d subnets from routing table:", len(subnets))
	for _, subnet := range subnets {
		t.Logf("  - %s via %v on %s", subnet.Subnet, subnet.Gateway, subnet.Interface)
	}
}
