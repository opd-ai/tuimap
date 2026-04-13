//go:build integration

// Package tracker provides device tracking and alert management.
// Integration tests in this file test end-to-end workflows.
package tracker

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opd-ai/tuimap/internal/scanner"
)

// TestIntegrationFullWorkflow tests a complete device tracking workflow.
func TestIntegrationFullWorkflow(t *testing.T) {
	// Create temporary directory for storage
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize storage
	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Create registry
	registry := NewRegistry(1 * time.Second)

	// Simulate device discovery
	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Hostname: "router.local",
			Ports:    []int{80, 443},
			Metadata: make(map[string]interface{}),
		},
		{
			IP:       net.ParseIP("192.168.1.100"),
			MAC:      net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			Hostname: "laptop.local",
			Ports:    []int{22},
			Metadata: make(map[string]interface{}),
		},
	}

	// Update registry
	err = registry.Update(devices)
	if err != nil {
		t.Fatalf("Failed to update registry: %v", err)
	}

	// Verify devices are tracked
	if registry.Count() != 2 {
		t.Errorf("Expected 2 devices, got %d", registry.Count())
	}

	// Verify alerts were generated
	time.Sleep(50 * time.Millisecond) // Allow alerts to propagate
	alerts := registry.GetAlerts()
	if len(alerts) < 2 {
		t.Errorf("Expected at least 2 new device alerts, got %d", len(alerts))
	}

	// Save devices to storage
	allDevices := registry.GetDevices()
	err = storage.SaveDevices(allDevices)
	if err != nil {
		t.Fatalf("Failed to save devices: %v", err)
	}

	// Save alerts to storage
	for _, alert := range alerts {
		err = storage.SaveAlert(alert)
		if err != nil {
			t.Fatalf("Failed to save alert: %v", err)
		}
	}

	// Verify persistence by loading from storage
	loadedDevices, err := storage.LoadDevices()
	if err != nil {
		t.Fatalf("Failed to load devices: %v", err)
	}

	if len(loadedDevices) != 2 {
		t.Errorf("Expected 2 loaded devices, got %d", len(loadedDevices))
	}

	loadedAlerts, err := storage.LoadAlerts()
	if err != nil {
		t.Fatalf("Failed to load alerts: %v", err)
	}

	if len(loadedAlerts) < 2 {
		t.Errorf("Expected at least 2 loaded alerts, got %d", len(loadedAlerts))
	}

	t.Logf("Integration test passed: %d devices, %d alerts", len(loadedDevices), len(loadedAlerts))
}

// TestIntegrationDeviceStateTransitions tests device state changes.
func TestIntegrationDeviceStateTransitions(t *testing.T) {
	registry := NewRegistry(100 * time.Millisecond) // Short threshold for testing

	// Initial discovery
	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.50"),
			Hostname: "test-device",
			Status:   scanner.StatusNew,
		},
	}

	registry.Update(devices)

	// Verify new device
	device, err := registry.GetDevice("192.168.1.50")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	if device.Status != scanner.StatusNew && device.Status != scanner.StatusOnline {
		t.Errorf("Expected StatusNew or StatusOnline, got %s", device.Status)
	}

	// Wait for offline threshold
	time.Sleep(200 * time.Millisecond)

	// Update without the device (simulates device going offline)
	registry.Update([]scanner.Device{})

	// Verify offline status
	device, err = registry.GetDevice("192.168.1.50")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	if device.Status != scanner.StatusOffline {
		t.Errorf("Expected StatusOffline, got %s", device.Status)
	}

	// Device comes back online
	registry.Update(devices)

	device, err = registry.GetDevice("192.168.1.50")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	if device.Status != scanner.StatusOnline {
		t.Errorf("Expected StatusOnline after reconnect, got %s", device.Status)
	}

	t.Log("State transitions test passed")
}

// TestIntegrationConcurrentUpdates tests concurrent registry updates.
func TestIntegrationConcurrentUpdates(t *testing.T) {
	registry := NewRegistry(5 * time.Minute)

	// Perform concurrent updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			devices := []scanner.Device{
				{
					IP:       net.ParseIP("192.168.1." + string(rune('0'+idx%10))),
					Hostname: "concurrent-test",
				},
			}
			registry.Update(devices)
			done <- true
		}(i)
	}

	// Wait for all updates
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify registry is in consistent state
	if registry.Count() == 0 {
		t.Error("Expected at least one device after concurrent updates")
	}

	t.Logf("Concurrent updates test passed: %d devices tracked", registry.Count())
}

// TestIntegrationStoragePersistence tests database persistence across restarts.
func TestIntegrationStoragePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persistence_test.db")

	// First session: create and save data
	func() {
		storage, err := NewStorage(dbPath, 24*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer func() { _ = storage.Close() }()

		device := scanner.Device{
			IP:       net.ParseIP("10.0.0.1"),
			MAC:      net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01},
			Hostname: "persistent-device",
			Metadata: make(map[string]interface{}),
		}

		err = storage.SaveDevice(device)
		if err != nil {
			t.Fatalf("Failed to save device: %v", err)
		}
	}()

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file should persist after close")
	}

	// Second session: load persisted data
	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to reopen storage: %v", err)
	}
	defer func() { _ = storage.Close() }()

	devices, err := storage.LoadDevices()
	if err != nil {
		t.Fatalf("Failed to load devices: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 persisted device, got %d", len(devices))
	}

	if devices[0].Hostname != "persistent-device" {
		t.Errorf("Expected hostname 'persistent-device', got '%s'", devices[0].Hostname)
	}

	t.Log("Storage persistence test passed")
}

// TestIntegrationExportFormats tests data export functionality.
func TestIntegrationExportFormats(t *testing.T) {
	registry := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{
			IP:       net.ParseIP("172.16.0.1"),
			MAC:      net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			Hostname: "export-test-device",
			Vendor:   "Test Vendor",
			Ports:    []int{80, 443, 8080},
		},
	}

	registry.Update(devices)

	// Test JSON export
	data, err := registry.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Export returned empty data")
	}

	// Verify JSON structure
	if data[0] != '{' && data[0] != '[' {
		t.Error("Export does not appear to be valid JSON")
	}

	t.Logf("Export test passed: %d bytes exported", len(data))
}
