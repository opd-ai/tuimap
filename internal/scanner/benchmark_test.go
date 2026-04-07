package scanner

import (
	"context"
	"net"
	"testing"
	"time"
)

// BenchmarkARPScanInit benchmarks ARP scanner initialization (requires interface).
func BenchmarkARPScanInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// NewARPScanner requires interface name, may fail without valid interface
		_, _ = NewARPScanner("lo", 256, 3*time.Second, 1)
	}
}

// BenchmarkICMPScanInit benchmarks ICMP scanner initialization (no network required).
func BenchmarkICMPScanInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewICMPScanner(256, 4*time.Second, 1)
	}
}

// BenchmarkTCPScanInit benchmarks TCP scanner initialization (no network required).
func BenchmarkTCPScanInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewTCPScanner(512, 500*time.Millisecond, []int{22, 80, 443, 3389, 8080})
	}
}

// BenchmarkOrchestratorInit benchmarks orchestrator initialization (no network required).
func BenchmarkOrchestratorInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOrchestrator(10 * time.Second)
	}
}

// BenchmarkMultiSubnetScannerInit benchmarks multi-subnet scanner initialization.
func BenchmarkMultiSubnetScannerInit(b *testing.B) {
	orch := NewOrchestrator(10 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMultiSubnetScanner(orch)
	}
}

// BenchmarkSubnetDiscovery benchmarks subnet discovery (no network required).
func BenchmarkSubnetDiscovery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DiscoverSubnets()
	}
}

// BenchmarkTCPScanLocalhost benchmarks TCP scanning localhost.
// This tests the scan overhead without network latency.
func BenchmarkTCPScanLocalhost(b *testing.B) {
	scanner := NewTCPScanner(10, 100*time.Millisecond, nil)
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")
	ports := []int{65432} // unlikely to be open

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanPorts(ctx, ip, ports)
	}
}

// BenchmarkOrchestratorFullScan benchmarks orchestrator with timeout.
// Note: This requires network access and may be slow. Use -benchtime=1x for single run.
// Target: <10s for /24 network.
func BenchmarkOrchestratorFullScan(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping full scan benchmark in short mode")
	}

	orch := NewOrchestrator(10 * time.Second)

	// Add TCP scanner only (doesn't require root)
	tcpScanner := NewTCPScanner(256, 500*time.Millisecond, []int{80})
	orch.AddScanner(tcpScanner)

	// Use a small subnet for benchmark (127.0.0.0/30 = 4 IPs)
	subnet := "127.0.0.0/30"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = orch.Scan(ctx, subnet)
		cancel()
	}
}

// BenchmarkResultMerging benchmarks device merging performance.
func BenchmarkResultMerging(b *testing.B) {
	now := time.Now()
	devices := make([]*Device, 256)
	for i := 0; i < 256; i++ {
		devices[i] = &Device{
			IP:       net.IPv4(192, 168, 1, byte(i)),
			MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, byte(i)},
			Ports:    []int{80, 443},
			LastSeen: now,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		merged := make(map[string]*Device)
		for _, device := range devices {
			key := device.IP.String()
			if _, exists := merged[key]; !exists {
				merged[key] = device
			}
		}
	}
}

// BenchmarkIPGeneration benchmarks IP address generation for /24.
func BenchmarkIPGeneration24(b *testing.B) {
	_, subnet, _ := net.ParseCIDR("192.168.1.0/24")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateIPs(subnet)
	}
}

// BenchmarkIPGeneration16 benchmarks IP address generation for /16.
func BenchmarkIPGeneration16(b *testing.B) {
	_, subnet, _ := net.ParseCIDR("192.168.0.0/16")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateIPs(subnet)
	}
}
