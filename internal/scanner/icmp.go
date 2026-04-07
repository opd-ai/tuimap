// Package scanner provides network scanning functionality for device discovery.
package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ICMPScanner implements ICMP ping-based network scanning.
// It uses ICMP echo requests for layer 3 device discovery.
type ICMPScanner struct {
	workers int
	timeout time.Duration
	count   int
}

// NewICMPScanner creates a new ICMP scanner.
func NewICMPScanner(workers int, timeout time.Duration, count int) *ICMPScanner {
	return &ICMPScanner{
		workers: workers,
		timeout: timeout,
		count:   count,
	}
}

// buildICMPEchoRequest creates an ICMP echo request message.
func buildICMPEchoRequest() ([]byte, error) {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   1,
			Seq:  1,
			Data: []byte("tuimap"),
		},
	}
	return msg.Marshal(nil)
}

// setConnDeadline sets the connection deadline from context or timeout.
func setConnDeadline(conn *icmp.PacketConn, ctx context.Context, timeout time.Duration) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(timeout)
	}
	conn.SetDeadline(deadline)
}

// isEchoReply checks if the ICMP message is an echo reply.
func isEchoReply(data []byte) bool {
	rm, err := icmp.ParseMessage(1, data)
	if err != nil {
		return false
	}
	return rm.Type == ipv4.ICMPTypeEchoReply
}

// Name returns the scanner name.
func (s *ICMPScanner) Name() string {
	return "icmp"
}

// Scan performs an ICMP ping scan on the given subnet.
func (s *ICMPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	// Generate list of IPs to scan
	ips := generateIPs(ipNet)
	if len(ips) == 0 {
		return nil, nil
	}

	// Result collection
	results := make(chan Device, len(ips))
	seen := &sync.Map{}

	// Create worker pool
	var wg sync.WaitGroup
	ipChan := make(chan net.IP, len(ips))

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.pingWorker(ctx, ipChan, results, seen)
		}()
	}

	// Feed IPs to workers
	go func() {
		for _, ip := range ips {
			select {
			case ipChan <- ip:
			case <-ctx.Done():
				return
			}
		}
		close(ipChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect results
	var devices []Device
	for device := range results {
		devices = append(devices, device)
	}

	return devices, nil
}

// pingWorker processes IPs from the channel and pings each one.
// It reuses a single ICMP connection across multiple hosts to avoid socket churn.
func (s *ICMPScanner) pingWorker(ctx context.Context, ips <-chan net.IP, results chan<- Device, seen *sync.Map) {
	// Try to create a privileged ICMP connection for this worker
	privConn, privErr := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if privErr == nil {
		defer privConn.Close()
	}

	// Create unprivileged connection as fallback
	unprivConn, unprivErr := icmp.ListenPacket("udp4", "")
	if unprivErr == nil {
		defer unprivConn.Close()
	}

	for ip := range ips {
		select {
		case <-ctx.Done():
			return
		default:
		}

		success := false
		if privConn != nil {
			success = s.pingWithConn(ctx, ip, privConn, true)
		}
		if !success && unprivConn != nil {
			success = s.pingWithConn(ctx, ip, unprivConn, false)
		}

		if success {
			// Deduplicate
			key := ip.String()
			if _, loaded := seen.LoadOrStore(key, true); loaded {
				continue
			}

			now := time.Now()
			device := Device{
				IP:        ip,
				LastSeen:  now,
				FirstSeen: now,
				Status:    StatusNew,
				Metadata:  make(map[string]interface{}),
			}

			select {
			case results <- device:
			case <-ctx.Done():
				return
			}
		}
	}
}

// ping sends ICMP echo requests to the target IP and returns true if it responds.
// This method creates new connections per host (used when worker-level conn reuse is not possible).
func (s *ICMPScanner) ping(ctx context.Context, ip net.IP) bool {
	// Try privileged ICMP first, fall back to UDP
	if s.pingPrivileged(ctx, ip) {
		return true
	}
	return s.pingUnprivileged(ctx, ip)
}

// pingWithConn sends ICMP echo requests using an existing connection.
func (s *ICMPScanner) pingWithConn(ctx context.Context, ip net.IP, conn *icmp.PacketConn, privileged bool) bool {
	setConnDeadline(conn, ctx, s.timeout)

	msgBytes, err := buildICMPEchoRequest()
	if err != nil {
		return false
	}

	var dst net.Addr
	if privileged {
		dst = &net.IPAddr{IP: ip}
	} else {
		dst = &net.UDPAddr{IP: ip, Port: 33434}
	}

	for i := 0; i < s.count; i++ {
		_, err = conn.WriteTo(msgBytes, dst)
		if err != nil {
			continue
		}

		reply := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(s.timeout))
		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			continue
		}

		// For privileged mode, verify response is from target
		if privileged && peer.String() != ip.String() {
			continue
		}

		if isEchoReply(reply[:n]) {
			return true
		}
	}

	return false
}

// pingPrivileged uses raw ICMP sockets (requires root).
func (s *ICMPScanner) pingPrivileged(ctx context.Context, ip net.IP) bool {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	defer conn.Close()

	return s.pingWithConn(ctx, ip, conn, true)
}

// pingUnprivileged uses UDP-based ICMP (no root required).
func (s *ICMPScanner) pingUnprivileged(ctx context.Context, ip net.IP) bool {
	conn, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		return false
	}
	defer conn.Close()

	return s.pingWithConn(ctx, ip, conn, false)
}
