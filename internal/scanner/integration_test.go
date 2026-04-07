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
