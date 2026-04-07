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

// NetcatTool implements netcat-like TCP/UDP client functionality.
type NetcatTool struct {
	timeout time.Duration
}

// NewNetcatTool creates a new netcat tool.
func NewNetcatTool(timeout time.Duration) *NetcatTool {
	return &NetcatTool{timeout: timeout}
}

// Name returns the tool name.
func (t *NetcatTool) Name() string {
	return "netcat"
}

// Validate validates the arguments.
func (t *NetcatTool) Validate(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: netcat <host> <port> [--udp] [--data <string>]")
	}
	_, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[1])
	}
	return nil
}

// netcatOptions holds parsed command-line options.
type netcatOptions struct {
	host   string
	port   string
	useUDP bool
	data   string
}

// Execute runs the netcat tool.
func (t *NetcatTool) Execute(ctx context.Context, args []string) (<-chan string, error) {
	if err := t.Validate(args); err != nil {
		return nil, err
	}

	opts := t.parseOptions(args)
	output := make(chan string, 100)

	go func() {
		defer close(output)
		t.runNetcat(ctx, opts, output)
	}()

	return output, nil
}

// parseOptions extracts netcat options from arguments.
func (t *NetcatTool) parseOptions(args []string) netcatOptions {
	opts := netcatOptions{host: args[0], port: args[1]}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--udp", "-u":
			opts.useUDP = true
		case "--data", "-d":
			if i+1 < len(args) {
				opts.data = args[i+1]
				i++
			}
		}
	}
	return opts
}

// runNetcat performs the connection and data transfer.
func (t *NetcatTool) runNetcat(ctx context.Context, opts netcatOptions, output chan<- string) {
	protocol := "tcp"
	if opts.useUDP {
		protocol = "udp"
	}

	addr := net.JoinHostPort(opts.host, opts.port)
	output <- fmt.Sprintf("Connecting to %s (%s)...\n", addr, protocol)

	conn, err := t.dial(ctx, protocol, addr)
	if err != nil {
		output <- fmt.Sprintf("Connection failed: %v\n", err)
		return
	}
	defer conn.Close()

	output <- fmt.Sprintf("Connected to %s\n", addr)

	if err := t.sendData(conn, opts.data, output); err != nil {
		return
	}
	t.readResponses(ctx, conn, output)
}

// dial establishes a connection with timeout.
func (t *NetcatTool) dial(ctx context.Context, protocol, addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: t.timeout}
	return dialer.DialContext(ctx, protocol, addr)
}

// sendData sends data to the connection if provided.
func (t *NetcatTool) sendData(conn net.Conn, data string, output chan<- string) error {
	if data == "" {
		return nil
	}
	if _, err := conn.Write([]byte(data + "\n")); err != nil {
		output <- fmt.Sprintf("Write error: %v\n", err)
		return err
	}
	output <- fmt.Sprintf("Sent: %s\n", data)
	return nil
}

// readResponses reads and outputs responses from the connection.
func (t *NetcatTool) readResponses(ctx context.Context, conn net.Conn, output chan<- string) {
	conn.SetReadDeadline(time.Now().Add(t.timeout))
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			output <- fmt.Sprintf("Received: %s\n", scanner.Text())
		}
	}
}

// TCPConnect performs a simple TCP connection test.
func (t *NetcatTool) TCPConnect(ctx context.Context, host string, port int) (bool, time.Duration, error) {
	start := time.Now()
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := net.Dialer{Timeout: t.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, 0, err
	}
	conn.Close()
	return true, time.Since(start), nil
}

// Banner attempts to grab the service banner.
func (t *NetcatTool) Banner(ctx context.Context, host string, port int) (string, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := net.Dialer{Timeout: t.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return "", nil // No banner available
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
