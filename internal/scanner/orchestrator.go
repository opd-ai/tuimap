// Package scanner provides network scanning functionality for device discovery.
package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jackpal/gateway"
)

// Orchestrator coordinates multiple scanning methods for comprehensive discovery.
// It runs ARP, ICMP, and TCP scans in parallel and aggregates results.
type Orchestrator struct {
	scanners []Scanner
	timeout  time.Duration
}

// NewOrchestrator creates a new scan orchestrator with the given timeout.
func NewOrchestrator(timeout time.Duration) *Orchestrator {
	return &Orchestrator{
		scanners: make([]Scanner, 0),
		timeout:  timeout,
	}
}

// AddScanner adds a scanner to the orchestrator.
func (o *Orchestrator) AddScanner(s Scanner) {
	o.scanners = append(o.scanners, s)
}

// Scan runs all configured scanners in parallel and merges results.
func (o *Orchestrator) Scan(ctx context.Context, subnet string) (*ScanResult, error) {
	startTime := time.Now()

	// Create scan context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	// Run all scanners in parallel
	type scanOutput struct {
		devices []Device
		method  string
		err     error
	}

	results := make(chan scanOutput, len(o.scanners))
	var wg sync.WaitGroup

	for _, scanner := range o.scanners {
		wg.Add(1)
		go func(s Scanner) {
			defer wg.Done()
			devices, err := s.Scan(scanCtx, subnet)
			results <- scanOutput{
				devices: devices,
				method:  s.Name(),
				err:     err,
			}
		}(scanner)
	}

	// Close results channel when all scanners complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and merge results
	deviceMap := make(map[string]*Device)
	var scanErrors []error

	for output := range results {
		if output.err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("%s: %w", output.method, output.err))
			continue
		}

		for _, device := range output.devices {
			key := deviceKey(&device)
			existing, exists := deviceMap[key]
			if exists {
				// Merge device information
				mergeDevices(existing, &device)
			} else {
				deviceCopy := device
				deviceMap[key] = &deviceCopy
			}
		}
	}

	// Convert map to slice
	devices := make([]Device, 0, len(deviceMap))
	for _, device := range deviceMap {
		devices = append(devices, *device)
	}

	// Build network metadata
	networkInfo := o.buildNetworkMetadata(subnet)

	return &ScanResult{
		Devices:     devices,
		ScanTime:    time.Since(startTime),
		Method:      "orchestrated",
		NetworkInfo: networkInfo,
	}, nil
}

// deviceKey returns a unique key for a device (prefer MAC, fall back to IP).
func deviceKey(d *Device) string {
	if d.MAC != nil && len(d.MAC) > 0 {
		return d.MAC.String()
	}
	return d.IP.String()
}

// mergeDevices merges information from src into dst.
func mergeDevices(dst, src *Device) {
	// Prefer non-nil MAC
	if dst.MAC == nil && src.MAC != nil {
		dst.MAC = src.MAC
	}

	// Prefer non-empty hostname
	if dst.Hostname == "" && src.Hostname != "" {
		dst.Hostname = src.Hostname
	}

	// Prefer non-empty vendor
	if dst.Vendor == "" && src.Vendor != "" {
		dst.Vendor = src.Vendor
	}

	// Merge ports
	for _, port := range src.Ports {
		dst.Ports = appendPort(dst.Ports, port)
	}

	// Update last seen if newer
	if src.LastSeen.After(dst.LastSeen) {
		dst.LastSeen = src.LastSeen
	}

	// Update first seen if older
	if src.FirstSeen.Before(dst.FirstSeen) {
		dst.FirstSeen = src.FirstSeen
	}

	// Merge metadata
	if dst.Metadata == nil {
		dst.Metadata = make(map[string]interface{})
	}
	for k, v := range src.Metadata {
		if _, exists := dst.Metadata[k]; !exists {
			dst.Metadata[k] = v
		}
	}
}

// buildNetworkMetadata extracts metadata about the scanned network.
func (o *Orchestrator) buildNetworkMetadata(subnet string) NetworkMetadata {
	_, ipNet, _ := net.ParseCIDR(subnet)

	meta := NetworkMetadata{
		Subnet: subnet,
	}

	// Try to detect gateway
	if gw, err := gateway.DiscoverGateway(); err == nil {
		meta.Gateway = gw
	}

	// Try to detect interface
	if ipNet != nil {
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipAddr, ok := addr.(*net.IPNet); ok {
					if ipNet.Contains(ipAddr.IP) {
						meta.Interface = iface.Name
						break
					}
				}
			}
		}
	}

	return meta
}

// DetectSubnet automatically detects the local subnet to scan.
func DetectSubnet() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				// Skip link-local addresses
				if ipNet.IP.IsLinkLocalUnicast() {
					continue
				}

				// Return the CIDR notation
				ones, _ := ipNet.Mask.Size()
				return fmt.Sprintf("%s/%d", ipNet.IP.Mask(ipNet.Mask), ones), nil
			}
		}
	}

	return "", fmt.Errorf("no suitable subnet found")
}

// CreateDefaultOrchestrator creates an orchestrator with default scanners.
func CreateDefaultOrchestrator(ifaceName string) (*Orchestrator, error) {
	orch := NewOrchestrator(10 * time.Second)

	// Add ARP scanner (fastest, requires root)
	arpScanner, err := NewARPScanner(ifaceName, 256, 100*time.Millisecond, 2)
	if err == nil {
		orch.AddScanner(arpScanner)
	}

	// Add ICMP scanner (cross-subnet, requires root for privileged mode)
	icmpScanner := NewICMPScanner(256, 1*time.Second, 1)
	orch.AddScanner(icmpScanner)

	// Add TCP scanner (works without root)
	tcpScanner := NewTCPScanner(512, 500*time.Millisecond, nil)
	orch.AddScanner(tcpScanner)

	return orch, nil
}
