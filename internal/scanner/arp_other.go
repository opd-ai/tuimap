// Package scanner provides network scanning functionality for device discovery.

//go:build !linux

package scanner

import (
	"context"
	"fmt"
	"time"
)

// NewARPScanner returns an error on non-Linux platforms since ARP scanning
// requires AF_PACKET raw sockets.
func NewARPScanner(ifaceName string, workers int, timeout time.Duration, retries int) (*ARPScanner, error) {
	return nil, fmt.Errorf("ARP scanning is not supported on this platform (requires Linux AF_PACKET sockets)")
}

// Scan returns an error on non-Linux platforms since ARP scanning
// requires AF_PACKET raw sockets.
func (s *ARPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	return nil, fmt.Errorf("ARP scanning is not supported on this platform (requires Linux AF_PACKET sockets)")
}
