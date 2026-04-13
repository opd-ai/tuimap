// Package tools provides integrated network diagnostic tools.
package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// WhoisTool implements WHOIS lookup functionality.
type WhoisTool struct {
	timeout time.Duration
}

// NewWhoisTool creates a new whois tool.
func NewWhoisTool(timeout time.Duration) *WhoisTool {
	return &WhoisTool{timeout: timeout}
}

// Name returns the tool name.
func (t *WhoisTool) Name() string {
	return "whois"
}

// Validate validates the arguments.
func (t *WhoisTool) Validate(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: whois <domain|ip> [--server <whois-server>]")
	}
	return nil
}

// Execute runs the whois tool.
func (t *WhoisTool) Execute(ctx context.Context, args []string) (<-chan string, error) {
	if err := t.Validate(args); err != nil {
		return nil, err
	}

	query, server := t.parseArgs(args)
	output := make(chan string, 100)

	go func() {
		defer close(output)
		t.runWhois(ctx, query, server, output)
	}()

	return output, nil
}

// parseArgs extracts query and optional server from arguments.
func (t *WhoisTool) parseArgs(args []string) (query, server string) {
	query = args[0]
	for i := 1; i < len(args); i++ {
		if args[i] == "--server" || args[i] == "-h" {
			if i+1 < len(args) {
				server = args[i+1]
			}
			break
		}
	}
	return query, server
}

// runWhois performs the WHOIS query and outputs results.
func (t *WhoisTool) runWhois(ctx context.Context, query, server string, output chan<- string) {
	if server == "" {
		server = t.getWhoisServer(query)
	}

	output <- fmt.Sprintf("Querying %s for %s...\n\n", server, query)

	result, err := t.query(ctx, server, query)
	if err != nil {
		output <- fmt.Sprintf("Error: %v\n", err)
		return
	}

	t.outputLines(ctx, result, output)
}

// outputLines sends result line by line to the output channel.
func (t *WhoisTool) outputLines(ctx context.Context, result string, output chan<- string) {
	scanner := bufio.NewScanner(strings.NewReader(result))
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			output <- scanner.Text() + "\n"
		}
	}
}

// getWhoisServer determines the appropriate WHOIS server for the query.
func (t *WhoisTool) getWhoisServer(query string) string {
	// Check if it's an IP address
	if ip := net.ParseIP(query); ip != nil {
		// Use ARIN for IP addresses (it will redirect if needed)
		return "whois.arin.net"
	}

	// Get TLD from domain
	parts := strings.Split(query, ".")
	if len(parts) < 2 {
		return "whois.iana.org"
	}

	tld := strings.ToLower(parts[len(parts)-1])

	// Common TLD WHOIS servers
	servers := map[string]string{
		"com":  "whois.verisign-grs.com",
		"net":  "whois.verisign-grs.com",
		"org":  "whois.pir.org",
		"info": "whois.afilias.net",
		"biz":  "whois.biz",
		"io":   "whois.nic.io",
		"co":   "whois.nic.co",
		"me":   "whois.nic.me",
		"us":   "whois.nic.us",
		"uk":   "whois.nic.uk",
		"de":   "whois.denic.de",
		"fr":   "whois.nic.fr",
		"eu":   "whois.eu",
		"ru":   "whois.tcinet.ru",
		"cn":   "whois.cnnic.cn",
		"jp":   "whois.jprs.jp",
		"au":   "whois.auda.org.au",
		"ca":   "whois.cira.ca",
		"in":   "whois.registry.in",
		"br":   "whois.registro.br",
	}

	if server, ok := servers[tld]; ok {
		return server
	}

	// Fallback to IANA
	return "whois.iana.org"
}

// query performs the WHOIS query.
func (t *WhoisTool) query(ctx context.Context, server, query string) (string, error) {
	// Add port if not specified
	if !strings.Contains(server, ":") {
		server = server + ":43"
	}

	// Connect with timeout
	dialer := net.Dialer{Timeout: t.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", server)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", server, err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadline for the entire operation
	if err := conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return "", fmt.Errorf("failed to set connection deadline: %w", err)
	}

	// Send query (CRLF terminated per RFC 3912)
	_, err = fmt.Fprintf(conn, "%s\r\n", query)
	if err != nil {
		return "", fmt.Errorf("failed to send query: %w", err)
	}

	// Read response
	var result strings.Builder
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		result.WriteString(line)
	}

	return result.String(), nil
}

// Lookup performs a WHOIS lookup and returns the result.
func (t *WhoisTool) Lookup(ctx context.Context, query string) (string, error) {
	server := t.getWhoisServer(query)
	return t.query(ctx, server, query)
}

// LookupIP performs a WHOIS lookup for an IP address.
func (t *WhoisTool) LookupIP(ctx context.Context, ip string) (string, error) {
	return t.query(ctx, "whois.arin.net:43", ip)
}

// LookupDomain performs a WHOIS lookup for a domain.
func (t *WhoisTool) LookupDomain(ctx context.Context, domain string) (string, error) {
	server := t.getWhoisServer(domain)
	return t.query(ctx, server, domain)
}
