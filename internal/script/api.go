// Package script provides the embedded Tengo scripting engine.
package script

import (
	"context"
	"net"
	"sync"
	"time"
)

// APIBridge provides the interface between Tengo scripts and TuiMap functionality.
type APIBridge struct {
	scanner      ScannerFunc
	pinger       PingerFunc
	portScanner  PortScannerFunc
	resolver     ResolverFunc
	alertHandler AlertHandler
	devices      DevicesProvider
	storage      *sync.Map
}

// ScannerFunc is a function that scans a subnet.
type ScannerFunc func(ctx context.Context, subnet string) ([]map[string]interface{}, error)

// PingerFunc is a function that pings a host.
type PingerFunc func(ctx context.Context, host string) (bool, time.Duration)

// PortScannerFunc is a function that scans ports on a host.
type PortScannerFunc func(ctx context.Context, host string, ports []int) []int

// ResolverFunc is a function that resolves a hostname.
type ResolverFunc func(ctx context.Context, hostname string) []string

// AlertHandler handles alerts from scripts.
type AlertHandler func(level, message string)

// DevicesProvider provides device information.
type DevicesProvider func() []map[string]interface{}

// NewAPIBridge creates a new API bridge with default implementations.
func NewAPIBridge() *APIBridge {
	return &APIBridge{
		storage:      &sync.Map{},
		scanner:      defaultScanner,
		pinger:       defaultPinger,
		portScanner:  defaultPortScanner,
		resolver:     defaultResolver,
		alertHandler: defaultAlertHandler,
		devices:      defaultDevicesProvider,
	}
}

// SetScanner sets the scanner function.
func (a *APIBridge) SetScanner(f ScannerFunc) {
	a.scanner = f
}

// SetPinger sets the pinger function.
func (a *APIBridge) SetPinger(f PingerFunc) {
	a.pinger = f
}

// SetPortScanner sets the port scanner function.
func (a *APIBridge) SetPortScanner(f PortScannerFunc) {
	a.portScanner = f
}

// SetResolver sets the resolver function.
func (a *APIBridge) SetResolver(f ResolverFunc) {
	a.resolver = f
}

// SetAlertHandler sets the alert handler.
func (a *APIBridge) SetAlertHandler(h AlertHandler) {
	a.alertHandler = h
}

// SetDevicesProvider sets the devices provider.
func (a *APIBridge) SetDevicesProvider(p DevicesProvider) {
	a.devices = p
}

// Scan performs a network scan.
func (a *APIBridge) Scan(subnet string) []map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := a.scanner(ctx, subnet)
	if err != nil {
		return nil
	}
	return result
}

// Ping pings a host.
func (a *APIBridge) Ping(host string) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.pinger(ctx, host)
}

// PortScan scans ports on a host.
func (a *APIBridge) PortScan(host string, ports []int) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.portScanner(ctx, host, ports)
}

// Resolve resolves a hostname to IP addresses.
func (a *APIBridge) Resolve(hostname string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.resolver(ctx, hostname)
}

// Alert sends an alert.
func (a *APIBridge) Alert(level, message string) {
	a.alertHandler(level, message)
}

// GetDevices returns all tracked devices.
func (a *APIBridge) GetDevices() []map[string]interface{} {
	return a.devices()
}

// Get retrieves a value from storage.
func (a *APIBridge) Get(key string) interface{} {
	value, ok := a.storage.Load(key)
	if !ok {
		return nil
	}
	return value
}

// Set stores a value in storage.
func (a *APIBridge) Set(key string, value interface{}) {
	a.storage.Store(key, value)
}

// Delete removes a value from storage.
func (a *APIBridge) Delete(key string) {
	a.storage.Delete(key)
}

// Default implementations

func defaultScanner(ctx context.Context, subnet string) ([]map[string]interface{}, error) {
	// Default implementation returns empty result
	return []map[string]interface{}{}, nil
}

func defaultPinger(ctx context.Context, host string) (bool, time.Duration) {
	// Simple TCP ping implementation
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host+":80", 2*time.Second)
	if err != nil {
		conn, err = net.DialTimeout("tcp", host+":443", 2*time.Second)
		if err != nil {
			return false, 0
		}
	}
	_ = conn.Close()
	return true, time.Since(start)
}

func defaultPortScanner(ctx context.Context, host string, ports []int) []int {
	var openPorts []int
	for _, port := range ports {
		select {
		case <-ctx.Done():
			return openPorts
		default:
		}

		addr := net.JoinHostPort(host, string(rune(port)))
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			openPorts = append(openPorts, port)
			_ = conn.Close()
		}
	}
	return openPorts
}

func defaultResolver(ctx context.Context, hostname string) []string {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return []string{}
	}
	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}
	return result
}

func defaultAlertHandler(level, message string) {
	// Default: just print to stdout
	println("[" + level + "] " + message)
}

func defaultDevicesProvider() []map[string]interface{} {
	return []map[string]interface{}{}
}
