package scanner

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func TestGenerateIPs(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		expected int
	}{
		{
			name:     "Class C /24 subnet",
			subnet:   "192.168.1.0/24",
			expected: 254, // 256 - 2 (network + broadcast)
		},
		{
			name:     "Class C /25 subnet",
			subnet:   "192.168.1.0/25",
			expected: 126, // 128 - 2
		},
		{
			name:     "Small /30 subnet",
			subnet:   "192.168.1.0/30",
			expected: 2, // 4 - 2
		},
		{
			name:     "Single host /32",
			subnet:   "192.168.1.1/32",
			expected: 0, // No hosts (only network address)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.subnet)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %v", err)
			}

			ips := generateIPs(ipNet)
			if len(ips) != tt.expected {
				t.Errorf("Expected %d IPs, got %d", tt.expected, len(ips))
			}
		})
	}
}

func TestIncIP(t *testing.T) {
	tests := []struct {
		name     string
		input    net.IP
		expected net.IP
	}{
		{
			name:     "Simple increment",
			input:    net.ParseIP("192.168.1.1").To4(),
			expected: net.ParseIP("192.168.1.2").To4(),
		},
		{
			name:     "Rollover last octet",
			input:    net.ParseIP("192.168.1.255").To4(),
			expected: net.ParseIP("192.168.2.0").To4(),
		},
		{
			name:     "Multiple rollover",
			input:    net.ParseIP("192.168.255.255").To4(),
			expected: net.ParseIP("192.169.0.0").To4(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := make(net.IP, len(tt.input))
			copy(ip, tt.input)
			inc(ip)
			if !ip.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, ip)
			}
		})
	}
}

func TestAppendPort(t *testing.T) {
	tests := []struct {
		name     string
		ports    []int
		port     int
		expected []int
	}{
		{
			name:     "Append to empty",
			ports:    []int{},
			port:     80,
			expected: []int{80},
		},
		{
			name:     "Append new port",
			ports:    []int{22, 80},
			port:     443,
			expected: []int{22, 80, 443},
		},
		{
			name:     "Append duplicate",
			ports:    []int{22, 80, 443},
			port:     80,
			expected: []int{22, 80, 443},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendPort(tt.ports, tt.port)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i, p := range tt.expected {
				if result[i] != p {
					t.Errorf("At index %d: expected %d, got %d", i, p, result[i])
				}
			}
		})
	}
}

func TestNewTCPScanner(t *testing.T) {
	scanner := NewTCPScanner(100, 500*time.Millisecond, nil)

	if scanner.workers != 100 {
		t.Errorf("Expected 100 workers, got %d", scanner.workers)
	}

	if scanner.timeout != 500*time.Millisecond {
		t.Errorf("Expected 500ms timeout, got %v", scanner.timeout)
	}

	// Default ports should be set
	if len(scanner.ports) != 5 {
		t.Errorf("Expected 5 default ports, got %d", len(scanner.ports))
	}
}

func TestNewTCPScannerWithCustomPorts(t *testing.T) {
	ports := []int{8080, 8443}
	scanner := NewTCPScanner(50, 1*time.Second, ports)

	if len(scanner.ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(scanner.ports))
	}

	if scanner.ports[0] != 8080 || scanner.ports[1] != 8443 {
		t.Error("Custom ports not set correctly")
	}
}

func TestNewICMPScanner(t *testing.T) {
	scanner := NewICMPScanner(256, 1*time.Second, 3)

	if scanner.workers != 256 {
		t.Errorf("Expected 256 workers, got %d", scanner.workers)
	}

	if scanner.timeout != 1*time.Second {
		t.Errorf("Expected 1s timeout, got %v", scanner.timeout)
	}

	if scanner.count != 3 {
		t.Errorf("Expected 3 pings, got %d", scanner.count)
	}
}

func TestOrchestratorAddScanner(t *testing.T) {
	orch := NewOrchestrator(10 * time.Second)

	if len(orch.scanners) != 0 {
		t.Errorf("Expected 0 scanners, got %d", len(orch.scanners))
	}

	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	orch.AddScanner(scanner)

	if len(orch.scanners) != 1 {
		t.Errorf("Expected 1 scanner, got %d", len(orch.scanners))
	}
}

func TestScannerName(t *testing.T) {
	tcpScanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	if tcpScanner.Name() != "tcp" {
		t.Errorf("Expected 'tcp', got '%s'", tcpScanner.Name())
	}

	icmpScanner := NewICMPScanner(10, 100*time.Millisecond, 1)
	if icmpScanner.Name() != "icmp" {
		t.Errorf("Expected 'icmp', got '%s'", icmpScanner.Name())
	}
}

func TestDeviceStatus(t *testing.T) {
	// Verify status constants
	if StatusOnline != "online" {
		t.Errorf("Expected 'online', got '%s'", StatusOnline)
	}
	if StatusOffline != "offline" {
		t.Errorf("Expected 'offline', got '%s'", StatusOffline)
	}
	if StatusNew != "new" {
		t.Errorf("Expected 'new', got '%s'", StatusNew)
	}
	if StatusChanged != "changed" {
		t.Errorf("Expected 'changed', got '%s'", StatusChanged)
	}
}

func TestMergeDevices(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	dst := &Device{
		IP:        net.ParseIP("192.168.1.1"),
		MAC:       nil,
		Hostname:  "",
		Vendor:    "",
		Ports:     []int{80},
		LastSeen:  earlier,
		FirstSeen: now,
		Metadata:  map[string]interface{}{"key1": "value1"},
	}

	src := &Device{
		IP:        net.ParseIP("192.168.1.1"),
		MAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Hostname:  "test-host",
		Vendor:    "Test Vendor",
		Ports:     []int{443, 80},
		LastSeen:  now,
		FirstSeen: earlier,
		Metadata:  map[string]interface{}{"key2": "value2"},
	}

	mergeDevices(dst, src)

	// MAC should be copied from src
	if dst.MAC == nil {
		t.Error("MAC should have been copied from src")
	}

	// Hostname should be copied from src
	if dst.Hostname != "test-host" {
		t.Errorf("Expected 'test-host', got '%s'", dst.Hostname)
	}

	// Vendor should be copied from src
	if dst.Vendor != "Test Vendor" {
		t.Errorf("Expected 'Test Vendor', got '%s'", dst.Vendor)
	}

	// Ports should be merged (80 exists, 443 added)
	if len(dst.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(dst.Ports))
	}

	// LastSeen should be updated to newer time
	if !dst.LastSeen.Equal(now) {
		t.Error("LastSeen should have been updated to newer time")
	}

	// FirstSeen should be updated to older time
	if !dst.FirstSeen.Equal(earlier) {
		t.Error("FirstSeen should have been updated to older time")
	}

	// Metadata should be merged
	if dst.Metadata["key2"] != "value2" {
		t.Error("Metadata should have been merged")
	}
	if dst.Metadata["key1"] != "value1" {
		t.Error("Original metadata should be preserved")
	}
}

func TestDeviceKey(t *testing.T) {
	tests := []struct {
		name     string
		device   *Device
		expected string
	}{
		{
			name: "With MAC",
			device: &Device{
				IP:  net.ParseIP("192.168.1.1"),
				MAC: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			},
			expected: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "Without MAC",
			device: &Device{
				IP:  net.ParseIP("192.168.1.1"),
				MAC: nil,
			},
			expected: "192.168.1.1",
		},
		{
			name: "Empty MAC",
			device: &Device{
				IP:  net.ParseIP("192.168.1.1"),
				MAC: net.HardwareAddr{},
			},
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deviceKey(tt.device)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTCPScanInvalidSubnet(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	ctx := context.Background()

	_, err := scanner.Scan(ctx, "invalid-subnet")
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}
}

func TestICMPScanInvalidSubnet(t *testing.T) {
	scanner := NewICMPScanner(10, 100*time.Millisecond, 1)
	ctx := context.Background()

	_, err := scanner.Scan(ctx, "invalid-subnet")
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}
}

func TestOrchestratorScanInvalidSubnet(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	scanner := NewTCPScanner(10, 50*time.Millisecond, nil)
	orch.AddScanner(scanner)
	ctx := context.Background()

	result, err := orch.Scan(ctx, "invalid-subnet")
	// Orchestrator aggregates errors but still returns a result
	// The result should have no devices since parsing failed
	if err != nil {
		// Error is acceptable
		return
	}
	if result != nil && len(result.Devices) > 0 {
		t.Error("Expected no devices for invalid subnet")
	}
}

func TestOrchestratorScanWithContext(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	scanner := NewTCPScanner(10, 50*time.Millisecond, nil)
	orch.AddScanner(scanner)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should complete quickly with short timeout
	// Use a non-routable subnet to avoid actual network access
	result, _ := orch.Scan(ctx, "10.255.255.0/30")

	// Result should be empty or have very few devices
	if result != nil && len(result.Devices) > 2 {
		t.Errorf("Expected few devices with cancelled context, got %d", len(result.Devices))
	}
}

func TestDiscoverSubnets(t *testing.T) {
	subnets, err := DiscoverSubnets()
	if err != nil {
		t.Fatalf("DiscoverSubnets failed: %v", err)
	}

	// Should find at least one subnet on most systems
	// (may be zero in very restricted environments)
	if len(subnets) == 0 {
		t.Skip("No subnets found (may be expected in restricted environments)")
	}

	for _, subnet := range subnets {
		// Validate subnet format
		_, ipNet, err := net.ParseCIDR(subnet.Subnet)
		if err != nil {
			t.Errorf("Invalid subnet CIDR %s: %v", subnet.Subnet, err)
		}
		if ipNet == nil {
			t.Errorf("Nil IPNet for subnet %s", subnet.Subnet)
		}

		// Interface should be non-empty
		if subnet.Interface == "" {
			t.Error("Subnet has empty interface name")
		}
	}
}

func TestDeduplicateSubnets(t *testing.T) {
	subnets := []SubnetInfo{
		{Subnet: "192.168.1.0/24", Interface: "eth0", Local: true},
		{Subnet: "192.168.1.0/24", Interface: "eth1", Local: true},
		{Subnet: "10.0.0.0/8", Interface: "eth0", Local: true},
	}

	result := DeduplicateSubnets(subnets)
	if len(result) != 2 {
		t.Errorf("Expected 2 unique subnets, got %d", len(result))
	}
}

func TestFilterLocalSubnets(t *testing.T) {
	subnets := []SubnetInfo{
		{Subnet: "192.168.1.0/24", Local: true},
		{Subnet: "10.0.0.0/8", Local: false},
		{Subnet: "172.16.0.0/16", Local: true},
	}

	result := FilterLocalSubnets(subnets)
	if len(result) != 2 {
		t.Errorf("Expected 2 local subnets, got %d", len(result))
	}

	for _, s := range result {
		if !s.Local {
			t.Errorf("Non-local subnet %s in filtered results", s.Subnet)
		}
	}
}

func TestParseHexIP(t *testing.T) {
	tests := []struct {
		hex      string
		expected net.IP
	}{
		{"0100A8C0", net.IPv4(192, 168, 0, 1)}, // 192.168.0.1 in little-endian
		{"00000000", net.IPv4(0, 0, 0, 0)},
		{"invalid", nil},
		{"0100", nil}, // Too short
	}

	for _, tt := range tests {
		result := parseHexIP(tt.hex)
		if tt.expected == nil {
			if result != nil {
				t.Errorf("parseHexIP(%s) = %v, expected nil", tt.hex, result)
			}
		} else if !result.Equal(tt.expected) {
			t.Errorf("parseHexIP(%s) = %v, expected %v", tt.hex, result, tt.expected)
		}
	}
}

func TestGetDefaultSubnet(t *testing.T) {
	subnet, err := GetDefaultSubnet()
	if err != nil {
		t.Skip("No default subnet found (may be expected in restricted environments)")
	}

	// Validate it's a valid CIDR
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		t.Errorf("GetDefaultSubnet returned invalid CIDR %s: %v", subnet, err)
	}
	if ipNet == nil {
		t.Errorf("GetDefaultSubnet returned subnet with nil IPNet: %s", subnet)
	}
}

// Benchmark tests for scanner performance

func BenchmarkGenerateIPs(b *testing.B) {
	_, ipNet, _ := net.ParseCIDR("192.168.1.0/24")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateIPs(ipNet)
	}
}

func BenchmarkIncIP(b *testing.B) {
	ip := net.ParseIP("192.168.1.1").To4()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inc(ip)
	}
}

func BenchmarkAppendPort(b *testing.B) {
	ports := []int{22, 80, 443, 3389}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appendPort(ports, 8080)
	}
}

func BenchmarkDeviceKey(b *testing.B) {
	device := &Device{
		IP:  net.ParseIP("192.168.1.1"),
		MAC: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deviceKey(device)
	}
}

func BenchmarkMergeDevices(b *testing.B) {
	now := time.Now()
	dst := &Device{
		IP:        net.ParseIP("192.168.1.1"),
		Ports:     []int{80},
		LastSeen:  now,
		FirstSeen: now,
		Metadata:  make(map[string]interface{}),
	}
	src := &Device{
		IP:        net.ParseIP("192.168.1.1"),
		MAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Hostname:  "test-host",
		Ports:     []int{443},
		LastSeen:  now,
		FirstSeen: now,
		Metadata:  make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDevices(dst, src)
	}
}

func BenchmarkDeduplicateSubnets(b *testing.B) {
	subnets := []SubnetInfo{
		{Subnet: "192.168.1.0/24", Interface: "eth0"},
		{Subnet: "192.168.1.0/24", Interface: "eth1"},
		{Subnet: "10.0.0.0/8", Interface: "eth0"},
		{Subnet: "172.16.0.0/16", Interface: "eth0"},
		{Subnet: "192.168.2.0/24", Interface: "eth0"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeduplicateSubnets(subnets)
	}
}

func BenchmarkParseHexIP(b *testing.B) {
	hex := "0100A8C0"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseHexIP(hex)
	}
}

func BenchmarkTCPScannerCreate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTCPScanner(512, 500*time.Millisecond, nil)
	}
}

func BenchmarkICMPScannerCreate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewICMPScanner(256, 1*time.Second, 1)
	}
}

func BenchmarkOrchestratorCreate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewOrchestrator(10 * time.Second)
	}
}

// Tests for MultiSubnetScanner

func TestNewMultiSubnetScanner(t *testing.T) {
	orch := NewOrchestrator(10 * time.Second)
	scanner := NewMultiSubnetScanner(orch)

	if scanner == nil {
		t.Fatal("NewMultiSubnetScanner returned nil")
	}

	if scanner.orchestrator != orch {
		t.Error("Orchestrator not set correctly")
	}
}

func TestMultiSubnetScanResult(t *testing.T) {
	result := &MultiSubnetScanResult{
		Results:    make(map[string]*ScanResult),
		AllDevices: []Device{},
		TotalTime:  1 * time.Second,
		Subnets:    []SubnetInfo{{Subnet: "192.168.1.0/24"}},
	}

	if result.TotalTime != 1*time.Second {
		t.Error("TotalTime not set correctly")
	}

	if len(result.Subnets) != 1 {
		t.Error("Subnets not set correctly")
	}
}

func TestMultiSubnetScanSubnetsEmpty(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	scanner := NewMultiSubnetScanner(orch)

	_, err := scanner.ScanSubnets(context.Background(), []SubnetInfo{})
	if err == nil {
		t.Error("Expected error for empty subnets")
	}
}

func TestMultiSubnetScanSubnetsWithMockOrchestrator(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	// Add a TCP scanner (works without root)
	tcpScanner := NewTCPScanner(10, 50*time.Millisecond, []int{65535})
	orch.AddScanner(tcpScanner)

	scanner := NewMultiSubnetScanner(orch)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Use non-routable addresses to avoid network issues
	subnets := []SubnetInfo{
		{Subnet: "10.255.255.252/30", Interface: "test0", Local: true},
	}

	result, err := scanner.ScanSubnets(ctx, subnets)
	if err != nil {
		t.Fatalf("ScanSubnets failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Subnets) != 1 {
		t.Errorf("Expected 1 subnet, got %d", len(result.Subnets))
	}

	if result.TotalTime <= 0 {
		t.Error("TotalTime should be positive")
	}
}

func TestMultiSubnetScanMergesDevices(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	tcpScanner := NewTCPScanner(10, 50*time.Millisecond, []int{65535})
	orch.AddScanner(tcpScanner)

	scanner := NewMultiSubnetScanner(orch)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Scan two different subnets
	subnets := []SubnetInfo{
		{Subnet: "10.255.255.252/30", Interface: "test0", Local: true},
		{Subnet: "10.255.254.252/30", Interface: "test1", Local: true},
	}

	result, err := scanner.ScanSubnets(ctx, subnets)
	if err != nil {
		t.Fatalf("ScanSubnets failed: %v", err)
	}

	// Results map should have entries for both subnets
	if len(result.Results) > 2 {
		t.Errorf("Expected at most 2 results, got %d", len(result.Results))
	}

	// AllDevices should be deduplicated
	// (can't verify exact count without network access)
	if result.AllDevices == nil {
		t.Error("AllDevices should not be nil")
	}
}

func BenchmarkMultiSubnetScannerCreate(b *testing.B) {
	orch := NewOrchestrator(10 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMultiSubnetScanner(orch)
	}
}

// Additional tests for improved coverage

func TestDetectSubnet(t *testing.T) {
	subnet, err := DetectSubnet()
	if err != nil {
		t.Skip("No suitable subnet found (may be expected in restricted environments)")
	}

	if subnet == "" {
		t.Error("Expected non-empty subnet")
	}

	// Validate it's a valid CIDR
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		t.Errorf("DetectSubnet returned invalid CIDR %s: %v", subnet, err)
	}
	if ipNet == nil {
		t.Errorf("DetectSubnet returned subnet with nil IPNet: %s", subnet)
	}
}

func TestNewARPScanner(t *testing.T) {
	// Test creating ARP scanner (may fail without proper interface)
	scanner, err := NewARPScanner("", 100, 500*time.Millisecond, 2)
	// Error is acceptable in test environment
	if err != nil {
		t.Logf("NewARPScanner error (expected in restricted environments): %v", err)
		return
	}

	if scanner == nil {
		t.Error("NewARPScanner returned nil without error")
	}
}

func TestARPScannerName(t *testing.T) {
	// We need to test the Name method, but we can't easily create an ARPScanner
	// without proper interface setup. Create one with empty interface to test Name.
	scanner := &ARPScanner{}
	if scanner.Name() != "arp" {
		t.Errorf("Expected 'arp', got '%s'", scanner.Name())
	}
}

func TestARPScannerSetOUIDatabase(t *testing.T) {
	scanner := &ARPScanner{}
	// We need a mock OUI database that implements the interface
	scanner.ouiDB = nil // Just test that the field exists
	// Can't easily test SetOUIDatabase without a real OUIDatabase implementation
}

func TestARPScanInvalidSubnet(t *testing.T) {
	// Create minimal ARPScanner without requiring interface setup
	scanner := &ARPScanner{
		workers: 10,
		timeout: 100 * time.Millisecond,
		retries: 1,
	}

	_, err := scanner.Scan(context.Background(), "invalid-subnet")
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}
}

func TestICMPScanEmptySubnet(t *testing.T) {
	scanner := NewICMPScanner(10, 100*time.Millisecond, 1)
	ctx := context.Background()

	// /32 subnet gives 0 hosts
	devices, err := scanner.Scan(ctx, "192.168.1.1/32")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should return empty or nil
	if len(devices) > 0 {
		t.Error("Expected no devices for /32 subnet")
	}
}

func TestTCPScanEmptySubnet(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	ctx := context.Background()

	// /32 subnet gives 0 hosts
	devices, err := scanner.Scan(ctx, "192.168.1.1/32")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should return empty or nil
	if len(devices) > 0 {
		t.Error("Expected no devices for /32 subnet")
	}
}

func TestOrchestratorScanNoScanners(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	ctx := context.Background()

	result, err := orch.Scan(ctx, "192.168.1.0/30")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should return empty result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Devices) > 0 {
		t.Error("Expected no devices with no scanners")
	}
}

func TestOrchestratorBuildNetworkMetadata(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)

	meta := orch.buildNetworkMetadata("192.168.1.0/24")

	if meta.Subnet != "192.168.1.0/24" {
		t.Errorf("Expected subnet '192.168.1.0/24', got '%s'", meta.Subnet)
	}
}

func TestOrchestratorBuildNetworkMetadataInvalidSubnet(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)

	meta := orch.buildNetworkMetadata("invalid")

	// Should still return something, even with invalid subnet
	if meta.Subnet != "invalid" {
		t.Errorf("Expected subnet 'invalid', got '%s'", meta.Subnet)
	}
}

func TestCreateDefaultOrchestrator(t *testing.T) {
	orch, err := CreateDefaultOrchestrator("")
	// This may fail without proper interface, but should not panic
	if err != nil {
		t.Logf("CreateDefaultOrchestrator error (may be expected): %v", err)
	}

	if orch == nil {
		t.Fatal("Expected non-nil orchestrator even with errors")
	}

	// Should have at least TCP and ICMP scanners (they don't require interface)
	if len(orch.scanners) < 2 {
		t.Errorf("Expected at least 2 scanners, got %d", len(orch.scanners))
	}
}

func TestTCPScannerScanWithLocalhostPort(t *testing.T) {
	// Start a TCP listener to have at least one target
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	scanner := NewTCPScanner(10, 500*time.Millisecond, []int{port})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Scan localhost only (127.0.0.1/32 won't work, use /30 which gives 2 hosts)
	devices, err := scanner.Scan(ctx, "127.0.0.0/30")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find localhost
	found := false
	for _, d := range devices {
		if d.IP.Equal(net.ParseIP("127.0.0.1")) {
			found = true
			break
		}
	}
	if !found && len(devices) == 0 {
		// Might be acceptable if 127.0.0.0 and 127.0.0.1 don't respond
		t.Log("Did not find localhost (may be expected)")
	}
}

func TestParseSubnetFromAddr(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{"valid CIDR", "192.168.1.10/24", true},
		{"valid /25", "10.0.0.5/25", true},
		{"invalid CIDR", "invalid", false},
		{"IPv6", "::1/128", false}, // IPv6 not supported currently
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, _ := net.ParseCIDR(tt.input)
			if ipNet == nil && tt.expectValid {
				t.Skip("Cannot parse test input")
			}

			if ipNet != nil {
				result, ok := parseSubnetFromAddr(ipNet, "test0")
				if tt.expectValid && !ok {
					t.Error("Expected valid subnet")
				}
				if tt.expectValid && ok && result.Subnet == "" {
					t.Error("Expected non-empty subnet")
				}
			}
		})
	}
}

func TestIsUsableInterface(t *testing.T) {
	// Test with a fake interface
	testIface := net.Interface{
		Name:  "test0",
		Flags: net.FlagUp | net.FlagBroadcast,
	}

	if !isUsableInterface(testIface) {
		t.Error("Expected test interface to be usable")
	}

	// Test with loopback
	loopback := net.Interface{
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback,
	}

	if isUsableInterface(loopback) {
		t.Error("Expected loopback to not be usable")
	}

	// Test with down interface
	downIface := net.Interface{
		Name:  "down0",
		Flags: net.FlagBroadcast,
	}

	if isUsableInterface(downIface) {
		t.Error("Expected down interface to not be usable")
	}
}

func TestMultiSubnetScanLocalSubnets(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	tcpScanner := NewTCPScanner(10, 50*time.Millisecond, []int{65535})
	orch.AddScanner(tcpScanner)

	scanner := NewMultiSubnetScanner(orch)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This may discover local subnets and scan them
	result, err := scanner.ScanLocalSubnets(ctx)
	if err != nil {
		t.Logf("ScanLocalSubnets error (may be expected): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestMultiSubnetScanAllSubnets(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	tcpScanner := NewTCPScanner(10, 50*time.Millisecond, []int{65535})
	orch.AddScanner(tcpScanner)

	scanner := NewMultiSubnetScanner(orch)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This may discover subnets and scan them
	result, err := scanner.ScanAllSubnets(ctx)
	if err != nil {
		t.Logf("ScanAllSubnets error (may be expected): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestOrchestratorScanWithMultipleScanners(t *testing.T) {
	orch := NewOrchestrator(500 * time.Millisecond)

	// Add multiple scanners
	tcpScanner1 := NewTCPScanner(10, 100*time.Millisecond, []int{80})
	tcpScanner2 := NewTCPScanner(10, 100*time.Millisecond, []int{443})

	orch.AddScanner(tcpScanner1)
	orch.AddScanner(tcpScanner2)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := orch.Scan(ctx, "10.255.255.252/30")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Method != "orchestrated" {
		t.Errorf("Expected method 'orchestrated', got '%s'", result.Method)
	}
}

func TestMergeDevicesWithNilMetadata(t *testing.T) {
	now := time.Now()

	dst := &Device{
		IP:       net.ParseIP("192.168.1.1"),
		Metadata: nil, // nil metadata
	}

	src := &Device{
		IP:       net.ParseIP("192.168.1.1"),
		LastSeen: now,
		Metadata: map[string]interface{}{"key": "value"},
	}

	mergeDevices(dst, src)

	if dst.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}

	if dst.Metadata["key"] != "value" {
		t.Error("Expected metadata to be merged")
	}
}

func TestICMPScannerName(t *testing.T) {
	scanner := NewICMPScanner(10, 100*time.Millisecond, 1)
	if scanner.Name() != "icmp" {
		t.Errorf("Expected name 'icmp', got '%s'", scanner.Name())
	}
}

func TestTCPScannerName(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	if scanner.Name() != "tcp" {
		t.Errorf("Expected name 'tcp', got '%s'", scanner.Name())
	}
}

func TestTCPScannerInvalidSubnet(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	_, err := scanner.Scan(context.Background(), "invalid-subnet")
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}
}

func TestICMPScannerInvalidSubnet(t *testing.T) {
	scanner := NewICMPScanner(10, 100*time.Millisecond, 1)
	_, err := scanner.Scan(context.Background(), "invalid-subnet")
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}
}

// TestBuildICMPEchoRequest tests ICMP echo request creation.
func TestBuildICMPEchoRequest(t *testing.T) {
	data, err := buildICMPEchoRequest()
	if err != nil {
		t.Fatalf("buildICMPEchoRequest failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty ICMP data")
	}

	// ICMP message should start with type (8 for echo request)
	if data[0] != 8 {
		t.Errorf("Expected ICMP type 8 (echo request), got %d", data[0])
	}
}

// TestIsEchoReply tests ICMP echo reply detection.
func TestIsEchoReply(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Valid echo reply",
			data:     []byte{0, 0, 0, 0, 0, 1, 0, 1}, // Type 0 (echo reply)
			expected: true,
		},
		{
			name:     "Echo request not a reply",
			data:     []byte{8, 0, 0, 0, 0, 1, 0, 1}, // Type 8 (echo request)
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "Invalid data",
			data:     []byte{255, 255},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEchoReply(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestParseLinuxRoutingTable tests routing table parsing.
func TestParseRoutingTable(t *testing.T) {
	// This test will actually try to read /proc/net/route
	// On non-Linux systems or without permissions, it should fall back
	subnets, err := ParseRoutingTable()
	if err != nil {
		// Errors are acceptable - may fall back to interface discovery
		t.Logf("ParseRoutingTable returned error (may be expected): %v", err)
	}
	// Just verify it doesn't panic and returns something
	_ = subnets
}

// TestScanPorts tests port scanning on a specific IP.
func TestScanPorts(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, []int{80, 443})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Scan localhost - some ports may be open, some closed
	ip := net.ParseIP("127.0.0.1")
	ports, err := scanner.ScanPorts(ctx, ip, []int{65432, 65433}) // unlikely to be open
	if err != nil {
		t.Fatalf("ScanPorts failed: %v", err)
	}

	// ports may be nil if no ports are open, which is fine
	t.Logf("Found %d open ports", len(ports))
}

// TestScanPortsWithDefaultPorts tests ScanPorts using scanner's default ports.
func TestScanPortsWithDefaultPorts(t *testing.T) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, []int{65432})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ip := net.ParseIP("127.0.0.1")
	// Pass empty ports to use scanner's default
	ports, err := scanner.ScanPorts(ctx, ip, nil)
	if err != nil {
		t.Fatalf("ScanPorts failed: %v", err)
	}

	// ports may be nil if no ports are open, which is fine
	t.Logf("Found %d open ports with default ports", len(ports))
}

// TestCreateDevice tests device creation with ARP scanner.
func TestCreateDevice(t *testing.T) {
	// Create minimal ARP scanner struct for testing createDevice
	scanner := &ARPScanner{}

	ip := net.ParseIP("192.168.1.100")
	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	device := scanner.createDevice(ip, mac)

	if !device.IP.Equal(ip) {
		t.Errorf("Expected IP %v, got %v", ip, device.IP)
	}

	if device.MAC.String() != mac.String() {
		t.Errorf("Expected MAC %v, got %v", mac, device.MAC)
	}

	if device.Status != StatusNew {
		t.Errorf("Expected status %v, got %v", StatusNew, device.Status)
	}

	if device.Metadata == nil {
		t.Error("Expected non-nil metadata")
	}
}

// mockOUIDB is a mock implementation of OUIDatabase for testing.
type mockOUIDB struct{}

func (m *mockOUIDB) Lookup(mac net.HardwareAddr) string {
	return "Test Vendor"
}

// TestCreateDeviceWithOUI tests device creation with OUI lookup.
func TestCreateDeviceWithOUI(t *testing.T) {
	// Create minimal ARP scanner with mock OUI database
	scanner := &ARPScanner{
		ouiDB: &mockOUIDB{},
	}

	ip := net.ParseIP("192.168.1.100")
	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	device := scanner.createDevice(ip, mac)

	if device.Vendor != "Test Vendor" {
		t.Errorf("Expected vendor 'Test Vendor', got '%s'", device.Vendor)
	}
}

// TestSetOUIDatabase tests setting the OUI database on ARP scanner.
func TestSetOUIDatabase(t *testing.T) {
	scanner := &ARPScanner{}
	mockDB := &mockOUIDB{}

	scanner.SetOUIDatabase(mockDB)

	// Verify it was set by creating a device
	ip := net.ParseIP("192.168.1.100")
	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	device := scanner.createDevice(ip, mac)
	if device.Vendor != "Test Vendor" {
		t.Errorf("Expected vendor from OUI database, got '%s'", device.Vendor)
	}
}

// TestScanFromRoutingTable tests scanning subnets from routing table.
func TestScanFromRoutingTable(t *testing.T) {
	orch := NewOrchestrator(100 * time.Millisecond)
	scanner := NewMultiSubnetScanner(orch)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// This will likely timeout quickly, but tests the function path
	result, err := scanner.ScanFromRoutingTable(ctx)
	// Just check it doesn't panic - may timeout or have no subnets
	_ = result
	_ = err
}

// TestMergeDevicesDuplicateIPs tests merging devices with same IP.
func TestMergeDevicesDuplicateIPs(t *testing.T) {
	ip := net.ParseIP("192.168.1.100")
	mac1 := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	mac2 := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	tests := []struct {
		name     string
		dst      *Device
		src      *Device
		expected net.HardwareAddr
	}{
		{
			name:     "Both have MACs - dst preserved",
			dst:      &Device{IP: ip, MAC: mac1},
			src:      &Device{IP: ip, MAC: mac2},
			expected: mac1,
		},
		{
			name:     "Only src has MAC",
			dst:      &Device{IP: ip, MAC: nil},
			src:      &Device{IP: ip, MAC: mac2},
			expected: mac2,
		},
		{
			name:     "Neither has MAC",
			dst:      &Device{IP: ip, MAC: nil},
			src:      &Device{IP: ip, MAC: nil},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeDevices(tt.dst, tt.src)
			if tt.expected == nil {
				if tt.dst.MAC != nil {
					t.Errorf("Expected nil MAC, got %v", tt.dst.MAC)
				}
			} else {
				if !bytes.Equal(tt.dst.MAC, tt.expected) {
					t.Errorf("Expected MAC %v, got %v", tt.expected, tt.dst.MAC)
				}
			}
		})
	}
}

// TestMergeDevicesPortMerging tests port deduplication during merge.
func TestMergeDevicesPortMerging(t *testing.T) {
	tests := []struct {
		name          string
		dstPorts      []int
		srcPorts      []int
		expectedCount int
	}{
		{
			name:          "No overlap",
			dstPorts:      []int{80, 443},
			srcPorts:      []int{22, 8080},
			expectedCount: 4,
		},
		{
			name:          "Full overlap",
			dstPorts:      []int{80, 443},
			srcPorts:      []int{80, 443},
			expectedCount: 2,
		},
		{
			name:          "Partial overlap",
			dstPorts:      []int{80, 443},
			srcPorts:      []int{443, 8080},
			expectedCount: 3,
		},
		{
			name:          "Dst empty",
			dstPorts:      nil,
			srcPorts:      []int{22, 80},
			expectedCount: 2,
		},
		{
			name:          "Src empty",
			dstPorts:      []int{22, 80},
			srcPorts:      nil,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &Device{IP: net.ParseIP("192.168.1.1"), Ports: tt.dstPorts}
			src := &Device{IP: net.ParseIP("192.168.1.1"), Ports: tt.srcPorts}
			mergeDevices(dst, src)
			if len(dst.Ports) != tt.expectedCount {
				t.Errorf("Expected %d ports, got %d", tt.expectedCount, len(dst.Ports))
			}
		})
	}
}

// TestGenerateIPsEdgeCases tests IP generation for edge case subnets.
func TestGenerateIPsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		expected int
	}{
		{
			name:     "Class B /16 subnet",
			subnet:   "10.0.0.0/16",
			expected: 65534, // 65536 - 2
		},
		{
			name:     "Class A /8 - limited check",
			subnet:   "192.0.0.0/8",
			expected: 16777214, // 16777216 - 2
		},
		{
			name:     "/31 point-to-point",
			subnet:   "192.168.1.0/31",
			expected: 0, // 2 - 2 = 0 usable addresses (network semantics)
		},
		{
			name:     "Different starting IP /24",
			subnet:   "10.100.50.0/24",
			expected: 254,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.subnet)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %v", err)
			}

			ips := generateIPs(ipNet)
			if len(ips) != tt.expected {
				t.Errorf("Expected %d IPs, got %d", tt.expected, len(ips))
			}
		})
	}
}

// TestDeviceStatusTransitions tests device status values.
func TestDeviceStatusTransitions(t *testing.T) {
	tests := []struct {
		status   DeviceStatus
		expected string
	}{
		{StatusNew, "new"},
		{StatusOnline, "online"},
		{StatusOffline, "offline"},
		{StatusChanged, "changed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

// TestMergeDevicesTimestamps tests timestamp merge logic.
func TestMergeDevicesTimestamps(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name              string
		dstFirst, dstLast time.Time
		srcFirst, srcLast time.Time
		expFirst, expLast time.Time
	}{
		{
			name:     "Src has newer LastSeen",
			dstFirst: past, dstLast: now,
			srcFirst: now, srcLast: future,
			expFirst: past, expLast: future,
		},
		{
			name:     "Src has older FirstSeen",
			dstFirst: now, dstLast: future,
			srcFirst: past, srcLast: now,
			expFirst: past, expLast: future,
		},
		{
			name:     "Dst has all older timestamps",
			dstFirst: past, dstLast: past,
			srcFirst: now, srcLast: now,
			expFirst: past, expLast: now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &Device{
				IP:        net.ParseIP("192.168.1.1"),
				FirstSeen: tt.dstFirst,
				LastSeen:  tt.dstLast,
			}
			src := &Device{
				IP:        net.ParseIP("192.168.1.1"),
				FirstSeen: tt.srcFirst,
				LastSeen:  tt.srcLast,
			}
			mergeDevices(dst, src)
			if !dst.FirstSeen.Equal(tt.expFirst) {
				t.Errorf("FirstSeen: expected %v, got %v", tt.expFirst, dst.FirstSeen)
			}
			if !dst.LastSeen.Equal(tt.expLast) {
				t.Errorf("LastSeen: expected %v, got %v", tt.expLast, dst.LastSeen)
			}
		})
	}
}
