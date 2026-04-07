package api

import (
	"net"
	"testing"
	"time"
)

func TestDeviceStatusConstants(t *testing.T) {
	tests := []struct {
		status   DeviceStatus
		expected string
	}{
		{StatusOnline, "online"},
		{StatusOffline, "offline"},
		{StatusNew, "new"},
		{StatusChanged, "changed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestAlertTypeConstants(t *testing.T) {
	tests := []struct {
		alertType AlertType
		expected  string
	}{
		{AlertNewDevice, "new_device"},
		{AlertDeviceOffline, "device_offline"},
		{AlertPortChange, "port_change"},
		{AlertMACConflict, "mac_conflict"},
	}

	for _, tt := range tests {
		if string(tt.alertType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.alertType)
		}
	}
}

func TestDeviceStruct(t *testing.T) {
	device := Device{
		IP:        net.ParseIP("192.168.1.1"),
		MAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Hostname:  "test-host",
		Vendor:    "Test Vendor",
		Ports:     []int{80, 443},
		LastSeen:  time.Now(),
		FirstSeen: time.Now().Add(-1 * time.Hour),
		Status:    StatusOnline,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	if device.IP.String() != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", device.IP)
	}

	if device.MAC.String() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("Expected MAC aa:bb:cc:dd:ee:ff, got %s", device.MAC)
	}

	if device.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", device.Hostname)
	}

	if device.Status != StatusOnline {
		t.Errorf("Expected status online, got %s", device.Status)
	}

	if len(device.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(device.Ports))
	}
}

func TestScanResultStruct(t *testing.T) {
	result := ScanResult{
		Devices: []Device{
			{IP: net.ParseIP("192.168.1.1")},
		},
		ScanTime: 5 * time.Second,
		Method:   "tcp",
		NetworkInfo: NetworkInfo{
			Subnet:    "192.168.1.0/24",
			Gateway:   net.ParseIP("192.168.1.1"),
			Interface: "eth0",
		},
	}

	if len(result.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(result.Devices))
	}

	if result.ScanTime != 5*time.Second {
		t.Errorf("Expected 5s scan time, got %v", result.ScanTime)
	}

	if result.NetworkInfo.Subnet != "192.168.1.0/24" {
		t.Errorf("Expected subnet 192.168.1.0/24, got %s", result.NetworkInfo.Subnet)
	}
}

func TestScanOptionsStruct(t *testing.T) {
	opts := ScanOptions{
		Subnet:     "192.168.1.0/24",
		Methods:    []string{"arp", "icmp", "tcp"},
		Timeout:    10 * time.Second,
		ARPWorkers: 256,
		TCPPorts:   []int{22, 80, 443},
	}

	if opts.Subnet != "192.168.1.0/24" {
		t.Errorf("Expected subnet 192.168.1.0/24, got %s", opts.Subnet)
	}

	if len(opts.Methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(opts.Methods))
	}

	if opts.ARPWorkers != 256 {
		t.Errorf("Expected 256 ARP workers, got %d", opts.ARPWorkers)
	}
}

func TestAlertStruct(t *testing.T) {
	alert := Alert{
		Type: AlertNewDevice,
		Device: Device{
			IP: net.ParseIP("192.168.1.100"),
		},
		Timestamp: time.Now(),
		Message:   "New device detected",
		Severity:  2,
	}

	if alert.Type != AlertNewDevice {
		t.Errorf("Expected alert type new_device, got %s", alert.Type)
	}

	if alert.Severity != 2 {
		t.Errorf("Expected severity 2, got %d", alert.Severity)
	}

	if alert.Message != "New device detected" {
		t.Errorf("Expected message 'New device detected', got '%s'", alert.Message)
	}
}

func TestScannerConfigStruct(t *testing.T) {
	cfg := ScannerConfig{
		Interface:    "eth0",
		ScanInterval: 60 * time.Second,
		Timeout:      10 * time.Second,
		Methods:      []string{"arp", "icmp"},
	}

	if cfg.Interface != "eth0" {
		t.Errorf("Expected interface eth0, got %s", cfg.Interface)
	}

	if cfg.Timeout != 10*time.Second {
		t.Errorf("Expected 10s timeout, got %v", cfg.Timeout)
	}
}

func TestAlertConfigStruct(t *testing.T) {
	cfg := AlertConfig{
		Enabled: true,
		Rules: []AlertRule{
			{Type: "new_device", Severity: 1, Action: "notify"},
		},
	}

	if !cfg.Enabled {
		t.Error("Expected alerts to be enabled")
	}

	if len(cfg.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(cfg.Rules))
	}

	if cfg.Rules[0].Type != "new_device" {
		t.Errorf("Expected rule type new_device, got %s", cfg.Rules[0].Type)
	}
}

func TestNetworkInfoStruct(t *testing.T) {
	info := NetworkInfo{
		Subnet:    "10.0.0.0/8",
		Gateway:   net.ParseIP("10.0.0.1"),
		Interface: "wlan0",
	}

	if info.Subnet != "10.0.0.0/8" {
		t.Errorf("Expected subnet 10.0.0.0/8, got %s", info.Subnet)
	}

	if info.Gateway.String() != "10.0.0.1" {
		t.Errorf("Expected gateway 10.0.0.1, got %s", info.Gateway)
	}

	if info.Interface != "wlan0" {
		t.Errorf("Expected interface wlan0, got %s", info.Interface)
	}
}
