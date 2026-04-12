// Package tools provides integrated network diagnostic tools.
package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// TelnetTool implements telnet client functionality.
type TelnetTool struct {
	timeout time.Duration
}

// NewTelnetTool creates a new telnet tool.
func NewTelnetTool(timeout time.Duration) *TelnetTool {
	return &TelnetTool{timeout: timeout}
}

// Name returns the tool name.
func (t *TelnetTool) Name() string {
	return "telnet"
}

// Validate validates the arguments.
func (t *TelnetTool) Validate(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: telnet <host> [port]")
	}
	if len(args) >= 2 {
		_, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid port: %s", args[1])
		}
	}
	return nil
}

// Execute runs the telnet tool.
func (t *TelnetTool) Execute(ctx context.Context, args []string) (<-chan string, error) {
	if err := t.Validate(args); err != nil {
		return nil, err
	}

	host, port := t.parseHostPort(args)
	output := make(chan string, 100)

	go func() {
		defer close(output)
		t.runTelnet(ctx, host, port, output)
	}()

	return output, nil
}

// parseHostPort extracts host and port from arguments.
func (t *TelnetTool) parseHostPort(args []string) (host, port string) {
	host = args[0]
	port = "23" // Default telnet port
	if len(args) >= 2 {
		port = args[1]
	}
	return host, port
}

// runTelnet performs the telnet connection and reads data.
func (t *TelnetTool) runTelnet(ctx context.Context, host, port string, output chan<- string) {
	addr := net.JoinHostPort(host, port)
	output <- fmt.Sprintf("Trying %s...\n", addr)

	conn, err := t.dialTelnet(ctx, addr)
	if err != nil {
		output <- fmt.Sprintf("telnet: Unable to connect to remote host: %v\n", err)
		return
	}
	defer func() { _ = conn.Close() }()

	output <- fmt.Sprintf("Connected to %s.\n", host)
	output <- "Escape character is '^]'.\n"

	go t.handleNegotiation(conn)
	t.readTelnetData(ctx, conn, output)
}

// dialTelnet establishes a TCP connection for telnet.
func (t *TelnetTool) dialTelnet(ctx context.Context, addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: t.timeout}
	return dialer.DialContext(ctx, "tcp", addr)
}

// readTelnetData reads data from connection and sends to output.
func (t *TelnetTool) readTelnetData(ctx context.Context, conn net.Conn, output chan<- string) {
	_ = conn.SetReadDeadline(time.Now().Add(t.timeout))
	reader := bufio.NewReader(conn)

	for {
		select {
		case <-ctx.Done():
			output <- "\nConnection closed.\n"
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				output <- "\nConnection closed by foreign host.\n"
				return
			}
			if cleaned := t.filterTelnet(line); cleaned != "" {
				output <- cleaned
			}
		}
	}
}

// handleNegotiation handles telnet option negotiation.
func (t *TelnetTool) handleNegotiation(conn net.Conn) {
	// Simple telnet negotiation - reject all options
	buf := make([]byte, 3)
	for {
		n, err := conn.Read(buf)
		if err != nil || n < 3 {
			return
		}

		if buf[0] == 0xFF { // IAC (Interpret As Command)
			response := make([]byte, 3)
			response[0] = 0xFF // IAC

			switch buf[1] {
			case 0xFB: // WILL - respond with DONT
				response[1] = 0xFE
				response[2] = buf[2]
				_, _ = conn.Write(response)
			case 0xFD: // DO - respond with WONT
				response[1] = 0xFC
				response[2] = buf[2]
				_, _ = conn.Write(response)
			case 0xFC, 0xFE: // WONT/DONT - ignore
				continue
			}
		}
	}
}

// filterTelnet removes telnet control sequences from text.
func (t *TelnetTool) filterTelnet(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0xFF && i+2 < len(s) {
			// Skip IAC sequence
			i += 2
			continue
		}
		if s[i] >= 32 || s[i] == '\n' || s[i] == '\r' || s[i] == '\t' {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

// Connect establishes a telnet connection and returns the banner.
func (t *TelnetTool) Connect(ctx context.Context, host string, port int) (string, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := net.Dialer{Timeout: t.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	// Handle negotiation in background
	go t.handleNegotiation(conn)

	// Read banner with timeout
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)

	return t.filterTelnet(string(buf[:n])), nil
}
