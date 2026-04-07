// Package scanner provides network scanning functionality for device discovery.
package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// tcpTarget represents an IP/port combination to scan.
type tcpTarget struct {
	ip   net.IP
	port int
}

// TCPScanner implements TCP connect-based port scanning.
// It discovers devices by attempting TCP connections to common ports.
type TCPScanner struct {
	workers int
	timeout time.Duration
	ports   []int
}

// NewTCPScanner creates a new TCP scanner.
func NewTCPScanner(workers int, timeout time.Duration, ports []int) *TCPScanner {
	if len(ports) == 0 {
		ports = []int{22, 80, 443, 3389, 5900}
	}
	return &TCPScanner{
		workers: workers,
		timeout: timeout,
		ports:   ports,
	}
}

// Name returns the scanner name.
func (s *TCPScanner) Name() string {
	return "tcp"
}

// Scan performs a TCP connect scan on the given subnet.
func (s *TCPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	ips := generateIPs(ipNet)
	if len(ips) == 0 {
		return nil, nil
	}

	targets := s.createTargetChannel(ctx, ips)
	deviceMap := s.runScanWorkers(ctx, targets)
	return s.collectDevices(deviceMap), nil
}

// createTargetChannel creates a channel of TCP targets from IPs and ports.
func (s *TCPScanner) createTargetChannel(ctx context.Context, ips []net.IP) <-chan tcpTarget {
	targets := make(chan tcpTarget, len(ips)*len(s.ports))
	go func() {
		defer close(targets)
		for _, ip := range ips {
			for _, port := range s.ports {
				select {
				case targets <- tcpTarget{ip: ip, port: port}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return targets
}

// runScanWorkers starts the worker pool and returns the device map.
func (s *TCPScanner) runScanWorkers(ctx context.Context, targets <-chan tcpTarget) *sync.Map {
	deviceMap := &sync.Map{}
	var wg sync.WaitGroup

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanWorker(ctx, targets, deviceMap)
		}()
	}
	wg.Wait()
	return deviceMap
}

// collectDevices converts a sync.Map to a slice of devices.
func (s *TCPScanner) collectDevices(deviceMap *sync.Map) []Device {
	var devices []Device
	deviceMap.Range(func(key, value interface{}) bool {
		if device, ok := value.(*Device); ok {
			devices = append(devices, *device)
		}
		return true
	})
	return devices
}

// scanWorker processes targets from the channel.
func (s *TCPScanner) scanWorker(ctx context.Context, targets <-chan tcpTarget, results *sync.Map) {
	for target := range targets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.tcpConnect(ctx, target.ip, target.port) {
			// Get or create device entry
			key := target.ip.String()
			deviceI, loaded := results.LoadOrStore(key, &Device{
				IP:        target.ip,
				Ports:     []int{target.port},
				LastSeen:  time.Now(),
				FirstSeen: time.Now(),
				Status:    StatusNew,
				Metadata:  make(map[string]interface{}),
			})

			if loaded {
				// Update existing device with new port
				device := deviceI.(*Device)
				device.Ports = appendPort(device.Ports, target.port)
				device.LastSeen = time.Now()
			}
		}
	}
}

// tcpConnect attempts a TCP connection to the target.
func (s *TCPScanner) tcpConnect(ctx context.Context, ip net.IP, port int) bool {
	addr := fmt.Sprintf("%s:%d", ip, port)

	// Use context deadline if set, otherwise use scanner timeout
	timeout := s.timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	dialer := net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// appendPort adds a port to the list if not already present.
func appendPort(ports []int, port int) []int {
	for _, p := range ports {
		if p == port {
			return ports
		}
	}
	return append(ports, port)
}

// ScanPorts scans specific ports on a single IP address.
func (s *TCPScanner) ScanPorts(ctx context.Context, ip net.IP, ports []int) ([]int, error) {
	if len(ports) == 0 {
		ports = s.ports
	}

	var openPorts []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			if s.tcpConnect(ctx, ip, p) {
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	return openPorts, nil
}
