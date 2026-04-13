// Package tools provides integrated network diagnostic tools.
package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// TracerouteTool implements traceroute functionality.
type TracerouteTool struct {
	maxHops int
	timeout time.Duration
}

// NewTracerouteTool creates a new traceroute tool.
func NewTracerouteTool(maxHops int, timeout time.Duration) *TracerouteTool {
	if maxHops <= 0 {
		maxHops = 30
	}
	return &TracerouteTool{
		maxHops: maxHops,
		timeout: timeout,
	}
}

// Name returns the tool name.
func (t *TracerouteTool) Name() string {
	return "traceroute"
}

// Validate validates the arguments.
func (t *TracerouteTool) Validate(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: traceroute <host> [--max-hops <n>]")
	}
	return nil
}

// Execute runs the traceroute tool.
func (t *TracerouteTool) Execute(ctx context.Context, args []string) (<-chan string, error) {
	if err := t.Validate(args); err != nil {
		return nil, err
	}

	host := args[0]
	maxHops := t.parseMaxHops(args)
	output := make(chan string, 100)

	go func() {
		defer close(output)
		t.runTraceroute(ctx, host, maxHops, output)
	}()

	return output, nil
}

// parseMaxHops extracts max-hops from arguments, returning default if not found.
func (t *TracerouteTool) parseMaxHops(args []string) int {
	for i := 1; i < len(args); i++ {
		if args[i] == "--max-hops" || args[i] == "-m" {
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					return n
				}
			}
		}
	}
	return t.maxHops
}

// runTraceroute performs the traceroute operation and outputs results.
func (t *TracerouteTool) runTraceroute(ctx context.Context, host string, maxHops int, output chan<- string) {
	destIP, err := t.resolveIPv4(host)
	if err != nil {
		output <- err.Error()
		return
	}

	output <- fmt.Sprintf("traceroute to %s (%s), %d hops max\n", host, destIP, maxHops)

	hops := t.trace(ctx, destIP, maxHops)
	for i, hop := range hops {
		select {
		case <-ctx.Done():
			return
		default:
			output <- hop.String(i + 1)
		}
	}
}

// resolveIPv4 resolves a hostname to its first IPv4 address.
func (t *TracerouteTool) resolveIPv4(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("traceroute: unknown host %s", host)
	}

	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
	}
	return nil, fmt.Errorf("traceroute: no IPv4 address found for %s", host)
}

// Hop represents a single hop in the traceroute.
type Hop struct {
	IP       net.IP
	Hostname string
	RTT      time.Duration
	Reached  bool
	Timeout  bool
}

// String formats the hop for display.
func (h *Hop) String(num int) string {
	if h.Timeout {
		return fmt.Sprintf("%2d  *\n", num)
	}

	name := h.IP.String()
	if h.Hostname != "" && h.Hostname != h.IP.String() {
		name = fmt.Sprintf("%s (%s)", h.Hostname, h.IP)
	}

	return fmt.Sprintf("%2d  %s  %v\n", num, name, h.RTT.Round(time.Microsecond))
}

// trace performs the traceroute using ICMP.
func (t *TracerouteTool) trace(ctx context.Context, dest net.IP, maxHops int) []Hop {
	hops := make([]Hop, 0, maxHops)

	// Try ICMP traceroute first, fall back to UDP if it fails
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// Fall back to UDP-based traceroute
		return t.traceUDP(ctx, dest, maxHops)
	}
	defer func() { _ = conn.Close() }()

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return hops
		default:
		}

		hop := t.probeICMP(conn, dest, ttl)
		hops = append(hops, hop)

		if hop.Reached {
			break
		}
	}

	return hops
}

// probeICMP sends an ICMP echo request with the specified TTL.
func (t *TracerouteTool) probeICMP(conn *icmp.PacketConn, dest net.IP, ttl int) Hop {
	// Set TTL
	if err := conn.IPv4PacketConn().SetTTL(ttl); err != nil {
		return Hop{Timeout: true}
	}

	// Create ICMP echo request
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   ttl,
			Seq:  ttl,
			Data: []byte("TRACEROUTE"),
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return Hop{Timeout: true}
	}

	// Send packet
	start := time.Now()
	dst := &net.IPAddr{IP: dest}
	_, err = conn.WriteTo(msgBytes, dst)
	if err != nil {
		return Hop{Timeout: true}
	}

	// Wait for response
	_ = conn.SetReadDeadline(time.Now().Add(t.timeout))
	reply := make([]byte, 1500)
	n, peer, err := conn.ReadFrom(reply)
	rtt := time.Since(start)

	if err != nil {
		return Hop{Timeout: true}
	}

	// Parse response
	rm, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		return Hop{Timeout: true}
	}

	peerAddr, ok := peer.(*net.IPAddr)
	if !ok {
		return Hop{Timeout: true}
	}

	hop := Hop{
		IP:  peerAddr.IP,
		RTT: rtt,
	}

	// Resolve hostname
	names, _ := net.LookupAddr(peerAddr.IP.String())
	if len(names) > 0 {
		hop.Hostname = strings.TrimSuffix(names[0], ".")
	}

	// Check if we reached the destination
	if rm.Type == ipv4.ICMPTypeEchoReply {
		hop.Reached = true
	}

	return hop
}

// traceUDP performs UDP-based traceroute (fallback for unprivileged users).
func (t *TracerouteTool) traceUDP(ctx context.Context, dest net.IP, maxHops int) []Hop {
	hops := make([]Hop, 0, maxHops)

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return hops
		default:
		}

		hop := t.probeUDP(dest, ttl)
		hops = append(hops, hop)

		if hop.Reached {
			break
		}
	}

	return hops
}

// probeUDP sends a UDP probe with the specified TTL.
func (t *TracerouteTool) probeUDP(dest net.IP, ttl int) Hop {
	// UDP probing uses high ports and expects ICMP responses
	// This is a simplified version that may not work in all environments
	addr := net.JoinHostPort(dest.String(), strconv.Itoa(33434+ttl))

	conn, err := net.DialTimeout("udp", addr, t.timeout)
	if err != nil {
		return Hop{Timeout: true}
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	_, _ = conn.Write([]byte("TRACE"))

	// Wait for timeout (no response expected, just measure RTT)
	time.Sleep(100 * time.Millisecond)
	rtt := time.Since(start)

	// Try to determine if we reached the destination
	if tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(dest.String(), "80"), 100*time.Millisecond); err == nil {
		_ = tcpConn.Close()
		return Hop{IP: dest, RTT: rtt, Reached: true}
	}

	return Hop{Timeout: true}
}

// Trace performs a traceroute and returns the hops.
func (t *TracerouteTool) Trace(ctx context.Context, host string) ([]Hop, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	var destIP net.IP
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			destIP = ip4
			break
		}
	}

	if destIP == nil {
		return nil, errors.New("no IPv4 address found")
	}

	return t.trace(ctx, destIP, t.maxHops), nil
}
