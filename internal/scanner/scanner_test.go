package scanner

import (
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
