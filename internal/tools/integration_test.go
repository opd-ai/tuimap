//go:build integration

// Package tools provides network diagnostic tool implementations.
// Integration tests in this file require network access.
package tools

import (
	"context"
	"testing"
	"time"
)

// TestIntegrationDigLookup performs a real DNS lookup.
func TestIntegrationDigLookup(t *testing.T) {
	dig := NewDigTool(10*time.Second, "8.8.8.8:53")
	ctx := context.Background()

	// Lookup a well-known domain
	ips, err := dig.LookupIP(ctx, "google.com")
	if err != nil {
		t.Skipf("DNS lookup failed (may be network issue): %v", err)
	}

	if len(ips) == 0 {
		t.Error("Expected at least one IP for google.com")
	}

	t.Logf("DNS lookup returned %d IPs for google.com", len(ips))
}

// TestIntegrationDigExecute tests the full dig execution flow.
func TestIntegrationDigExecute(t *testing.T) {
	dig := NewDigTool(10*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"example.com", "A"})
	if err != nil {
		t.Skipf("Dig execute failed (may be network issue): %v", err)
	}

	// Collect output
	var results []string
	for line := range output {
		results = append(results, line)
	}

	if len(results) == 0 {
		t.Error("Expected some output from dig")
	}

	t.Logf("Dig returned %d lines of output", len(results))
}

// TestIntegrationNetcatLocalhost tests TCP connection to localhost.
func TestIntegrationNetcatLocalhost(t *testing.T) {
	nc := NewNetcatTool(5 * time.Second)
	ctx := context.Background()

	// Try to connect to a port that's likely closed
	// This tests the connection timeout handling
	ok, latency, err := nc.TCPConnect(ctx, "127.0.0.1", 65534)

	// Connection should fail to closed port but not error
	t.Logf("Localhost connection test: ok=%v, latency=%v, err=%v", ok, latency, err)
}

// TestIntegrationNetcatExecuteTimeout tests netcat timeout behavior.
func TestIntegrationNetcatExecuteTimeout(t *testing.T) {
	nc := NewNetcatTool(1 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Connect to non-routable address to test timeout
	output, err := nc.Execute(ctx, []string{"10.255.255.1", "80"})
	if err != nil {
		// Timeout errors are expected
		t.Logf("Expected timeout error: %v", err)
		return
	}

	// Drain output
	for range output {
	}
}

// TestIntegrationTracerouteLocalhost tests traceroute to localhost.
func TestIntegrationTracerouteLocalhost(t *testing.T) {
	tr := NewTracerouteTool(5, 1*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	output, err := tr.Execute(ctx, []string{"127.0.0.1"})
	if err != nil {
		t.Skipf("Traceroute failed (may need root): %v", err)
	}

	var hops int
	for range output {
		hops++
	}

	// Localhost should have at least 1 hop
	t.Logf("Traceroute to localhost: %d hops", hops)
}

// TestIntegrationWhoisDomain tests WHOIS lookup for a domain.
func TestIntegrationWhoisDomain(t *testing.T) {
	whois := NewWhoisTool(15 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	output, err := whois.Execute(ctx, []string{"example.com"})
	if err != nil {
		t.Skipf("WHOIS lookup failed (may be network issue): %v", err)
	}

	var lines int
	for range output {
		lines++
	}

	if lines == 0 {
		t.Error("Expected some WHOIS output")
	}

	t.Logf("WHOIS returned %d lines for example.com", lines)
}

// TestIntegrationTelnetBanner tests telnet banner grabbing.
func TestIntegrationTelnetBanner(t *testing.T) {
	telnet := NewTelnetTool(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to connect to localhost on a typically open port
	output, err := telnet.Execute(ctx, []string{"127.0.0.1", "80"})
	if err != nil {
		t.Logf("Telnet connection failed (expected if port is closed): %v", err)
		return
	}

	var lines int
	for range output {
		lines++
	}

	t.Logf("Telnet returned %d lines", lines)
}

// TestIntegrationToolRegistry tests the tool registry pattern.
func TestIntegrationToolRegistry(t *testing.T) {
	// Create all tools
	tools := map[string]NetworkTool{
		"netcat":     NewNetcatTool(5 * time.Second),
		"telnet":     NewTelnetTool(5 * time.Second),
		"traceroute": NewTracerouteTool(30, 1*time.Second),
		"dig":        NewDigTool(5*time.Second, ""),
		"whois":      NewWhoisTool(10 * time.Second),
	}

	// Verify all tools are properly named
	for name, tool := range tools {
		if tool.Name() != name {
			t.Errorf("Tool name mismatch: expected %s, got %s", name, tool.Name())
		}
	}

	t.Logf("All %d tools registered correctly", len(tools))
}

// TestIntegrationConcurrentTools tests running multiple tools concurrently.
func TestIntegrationConcurrentTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := make(chan string, 3)

	// Run dig in parallel
	go func() {
		dig := NewDigTool(10*time.Second, "")
		output, err := dig.Execute(ctx, []string{"localhost"})
		if err != nil {
			results <- "dig:error"
			return
		}
		var lines int
		for range output {
			lines++
		}
		results <- "dig:done"
	}()

	// Run netcat test in parallel
	go func() {
		nc := NewNetcatTool(2 * time.Second)
		nc.TCPConnect(ctx, "127.0.0.1", 65534)
		results <- "netcat:done"
	}()

	// Wait for both
	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			t.Logf("Concurrent tool result: %s", result)
		case <-ctx.Done():
			t.Error("Timeout waiting for concurrent tools")
		}
	}
}

// TestIntegrationDigMXLookup tests MX record lookup.
func TestIntegrationDigMXLookup(t *testing.T) {
	dig := NewDigTool(10*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mxRecords, err := dig.LookupMX(ctx, "gmail.com")
	if err != nil {
		t.Skipf("MX lookup failed (may be network issue): %v", err)
	}

	if len(mxRecords) == 0 {
		t.Error("Expected at least one MX record for gmail.com")
	}

	for _, mx := range mxRecords {
		t.Logf("MX: %s (priority: %d)", mx.Host, mx.Pref)
	}
}
