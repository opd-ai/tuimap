// Package scanner provides network scanning functionality for device discovery.
// This package implements multiple scanning methods (ARP, ICMP, TCP) to discover
// all devices on local networks in under 10 seconds.
package scanner

import (
	"context"
	"net"
	"time"
)

// Device represents a discovered network device
type Device struct {
	IP        net.IP
	MAC       net.HardwareAddr
	Hostname  string
	Vendor    string
	Ports     []int
	LastSeen  time.Time
	FirstSeen time.Time
	Status    DeviceStatus
	Metadata  map[string]interface{}
}

// DeviceStatus represents the current status of a device
type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
	StatusNew     DeviceStatus = "new"
	StatusChanged DeviceStatus = "changed"
)

// ScanResult contains the results of a network scan
type ScanResult struct {
	Devices     []Device
	ScanTime    time.Duration
	Method      string
	NetworkInfo NetworkMetadata
}

// NetworkMetadata contains metadata about the scanned network
type NetworkMetadata struct {
	Subnet    string
	Gateway   net.IP
	Interface string
}

// Scanner defines the interface for network scanners
type Scanner interface {
	// Scan performs a network scan on the given subnet
	Scan(ctx context.Context, subnet string) ([]Device, error)

	// Name returns the scanner name
	Name() string
}

// TODO: Implement ARP scanner
// TODO: Implement ICMP scanner
// TODO: Implement TCP port scanner
// TODO: Implement passive discovery
