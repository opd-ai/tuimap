// Package scanner provides network scanning functionality for device discovery.

//go:build !linux

package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ARPScanner implements ARP-based network scanning for device discovery.
// On non-Linux platforms, ARP scanning is not supported because it requires
// AF_PACKET raw sockets which are only available on Linux.
type ARPScanner struct {
	iface     *net.Interface
	workers   int
	timeout   time.Duration
	retries   int
	localIP   net.IP
	localMAC  net.HardwareAddr
	ouiDB     OUIDatabase
	ouiDBOnce sync.Once
}

// OUIDatabase provides MAC vendor lookup functionality.
type OUIDatabase interface {
	// Lookup returns the vendor name for a MAC address prefix.
	Lookup(mac net.HardwareAddr) string
}

// NewARPScanner returns an error on non-Linux platforms since ARP scanning
// requires AF_PACKET raw sockets.
func NewARPScanner(ifaceName string, workers int, timeout time.Duration, retries int) (*ARPScanner, error) {
	return nil, fmt.Errorf("ARP scanning is not supported on this platform (requires Linux AF_PACKET sockets)")
}

// getDefaultInterface returns the default network interface.
func getDefaultInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if interface has an IPv4 address
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				return &iface, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable network interface found")
}

// getInterfaceIP returns the IPv4 address of the interface.
func getInterfaceIP(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			return ipNet.IP.To4(), nil
		}
	}

	return nil, fmt.Errorf("no IPv4 address found on interface %s", iface.Name)
}

// generateIPs generates all IP addresses in a subnet (excluding network and broadcast).
func generateIPs(ipNet *net.IPNet) []net.IP {
	var ips []net.IP

	// Get subnet mask size
	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones

	// Calculate number of hosts (excluding network and broadcast)
	numHosts := (1 << hostBits) - 2
	if numHosts <= 0 {
		return nil
	}

	// Start from first host address
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)

	// Increment to first host
	inc(ip)

	for i := 0; i < numHosts; i++ {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		ips = append(ips, newIP)
		inc(ip)
	}

	return ips
}

// inc increments an IP address by 1.
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// collectDevices drains a device channel into a slice.
func collectDevices(ch <-chan Device) []Device {
	var devices []Device
	for device := range ch {
		devices = append(devices, device)
	}
	return devices
}
