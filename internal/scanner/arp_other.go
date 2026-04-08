// Package scanner provides network scanning functionality for device discovery.

//go:build !linux

package scanner

import (
	"fmt"
	"time"
)

// NewARPScanner returns an error on non-Linux platforms since ARP scanning
// requires AF_PACKET raw sockets.
func NewARPScanner(ifaceName string, workers int, timeout time.Duration, retries int) (*ARPScanner, error) {
	return nil, fmt.Errorf("ARP scanning is not supported on this platform (requires Linux AF_PACKET sockets)")
}
