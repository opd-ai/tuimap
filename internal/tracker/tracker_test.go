package tracker

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opd-ai/tuimap/internal/scanner"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	if r == nil {
		t.Fatal("Expected non-nil registry")
	}

	if r.Count() != 0 {
		t.Errorf("Expected empty registry, got %d devices", r.Count())
	}
}

func TestRegistryUpdate(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Hostname: "test-host",
			Ports:    []int{80, 443},
		},
	}

	err := r.Update(devices)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("Expected 1 device, got %d", r.Count())
	}
}

func TestRegistryGetDevice(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	ip := net.ParseIP("192.168.1.1")
	devices := []scanner.Device{
		{
			IP:       ip,
			Hostname: "test-host",
		},
	}

	r.Update(devices)

	device, err := r.GetDevice("192.168.1.1")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	if device.Hostname != "test-host" {
		t.Errorf("Expected 'test-host', got '%s'", device.Hostname)
	}
}

func TestRegistryGetDeviceNotFound(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	_, err := r.GetDevice("192.168.1.1")
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
}

func TestRegistryAlertOnNewDevice(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{
			IP:    net.ParseIP("192.168.1.1"),
			MAC:   net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Ports: []int{80},
		},
	}

	r.Update(devices)

	// Give alert channel time to receive
	time.Sleep(10 * time.Millisecond)

	alerts := r.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alert for new device")
		return
	}

	if alerts[0].Type != AlertNewDevice {
		t.Errorf("Expected AlertNewDevice, got %s", alerts[0].Type)
	}
}

func TestRegistryDeviceStatusChange(t *testing.T) {
	r := NewRegistry(1 * time.Millisecond) // Very short offline threshold

	// First update - device online
	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "test-host",
		},
	}
	r.Update(devices)

	// Get initial alert
	r.GetAlerts()

	// Wait for offline threshold
	time.Sleep(10 * time.Millisecond)

	// Update without the device
	r.Update([]scanner.Device{})

	// Check device is marked offline
	device, _ := r.GetDevice("192.168.1.1")
	if device.Status != scanner.StatusOffline {
		t.Errorf("Expected StatusOffline, got %s", device.Status)
	}
}

func TestRegistryPortChange(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	// First update with port 80
	devices := []scanner.Device{
		{
			IP:    net.ParseIP("192.168.1.1"),
			Ports: []int{80},
		},
	}
	r.Update(devices)
	r.GetAlerts() // Clear initial alert

	// Second update with different ports
	devices = []scanner.Device{
		{
			IP:    net.ParseIP("192.168.1.1"),
			Ports: []int{80, 443, 22},
		},
	}
	r.Update(devices)

	// Give alert channel time
	time.Sleep(10 * time.Millisecond)

	alerts := r.GetAlerts()
	foundPortChange := false
	for _, alert := range alerts {
		if alert.Type == AlertPortChange {
			foundPortChange = true
			break
		}
	}

	if !foundPortChange {
		t.Error("Expected AlertPortChange for port change")
	}
}

func TestRegistryOnlineCount(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1")},
		{IP: net.ParseIP("192.168.1.2")},
		{IP: net.ParseIP("192.168.1.3")},
	}
	r.Update(devices)

	if r.OnlineCount() != 3 {
		t.Errorf("Expected 3 online devices, got %d", r.OnlineCount())
	}
}

func TestRegistryClear(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1")},
		{IP: net.ParseIP("192.168.1.2")},
	}
	r.Update(devices)

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Expected 0 devices after clear, got %d", r.Count())
	}
}

func TestRegistryExport(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "test-host",
		},
	}
	r.Update(devices)

	data, err := r.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty export data")
	}
}

func TestAlertSeverity(t *testing.T) {
	tests := []struct {
		alertType AlertType
		expected  int
	}{
		{AlertMACConflict, 3},
		{AlertNewDevice, 2},
		{AlertDeviceOffline, 1},
		{AlertPortChange, 2},
		{AlertType("unknown"), 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.alertType), func(t *testing.T) {
			severity := getSeverity(tt.alertType)
			if severity != tt.expected {
				t.Errorf("Expected severity %d, got %d", tt.expected, severity)
			}
		})
	}
}

func TestStorageCreateAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestStorageSaveAndLoadDevice(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	device := scanner.Device{
		IP:       net.ParseIP("192.168.1.1"),
		MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Hostname: "test-host",
		Ports:    []int{80, 443},
		Metadata: make(map[string]interface{}),
	}

	err = storage.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	devices, err := storage.LoadDevices()
	if err != nil {
		t.Fatalf("Failed to load devices: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}

	loaded := devices[0]
	if loaded.IP.String() != device.IP.String() {
		t.Errorf("IP mismatch: expected %s, got %s", device.IP, loaded.IP)
	}
	if loaded.Hostname != device.Hostname {
		t.Errorf("Hostname mismatch: expected %s, got %s", device.Hostname, loaded.Hostname)
	}
}

func TestStorageSaveAndLoadAlert(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	alert := Alert{
		Type: AlertNewDevice,
		Device: scanner.Device{
			IP:       net.ParseIP("192.168.1.1"),
			Metadata: make(map[string]interface{}),
		},
		Timestamp: time.Now(),
		Message:   "Test alert",
		Severity:  2,
	}

	err = storage.SaveAlert(alert)
	if err != nil {
		t.Fatalf("Failed to save alert: %v", err)
	}

	alerts, err := storage.LoadAlerts()
	if err != nil {
		t.Fatalf("Failed to load alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(alerts))
	}

	loaded := alerts[0]
	if loaded.Type != alert.Type {
		t.Errorf("Type mismatch: expected %s, got %s", alert.Type, loaded.Type)
	}
	if loaded.Message != alert.Message {
		t.Errorf("Message mismatch: expected %s, got %s", alert.Message, loaded.Message)
	}
}

func TestStorageSaveDevices(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Metadata: make(map[string]interface{})},
		{IP: net.ParseIP("192.168.1.2"), Metadata: make(map[string]interface{})},
		{IP: net.ParseIP("192.168.1.3"), Metadata: make(map[string]interface{})},
	}

	err = storage.SaveDevices(devices)
	if err != nil {
		t.Fatalf("Failed to save devices: %v", err)
	}

	loaded, err := storage.LoadDevices()
	if err != nil {
		t.Fatalf("Failed to load devices: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(loaded))
	}
}

func TestRegistryDroppedAlerts(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	// Initial count should be 0
	if r.DroppedAlerts() != 0 {
		t.Errorf("Expected 0 dropped alerts initially, got %d", r.DroppedAlerts())
	}

	testDevice := scanner.Device{IP: net.ParseIP("192.168.1.1")}

	// Fill the alert channel (capacity 100)
	for i := 0; i < 100; i++ {
		r.triggerAlert(AlertNewDevice, testDevice, "test")
	}

	// The next alerts should be dropped
	r.triggerAlert(AlertNewDevice, testDevice, "dropped")
	r.triggerAlert(AlertNewDevice, scanner.Device{IP: net.ParseIP("192.168.1.2")}, "dropped")

	if r.DroppedAlerts() != 2 {
		t.Errorf("Expected 2 dropped alerts, got %d", r.DroppedAlerts())
	}
}

// Benchmark tests for tracker performance

func BenchmarkRegistryUpdate(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), MAC: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}},
		{IP: net.ParseIP("192.168.1.2"), MAC: net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Update(devices)
	}
}

func BenchmarkRegistryGetDevice(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1")},
	}
	r.Update(devices)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.GetDevice("192.168.1.1")
	}
}

func BenchmarkRegistryGetDevices(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := make([]scanner.Device, 100)
	for i := 0; i < 100; i++ {
		devices[i] = scanner.Device{
			IP: net.ParseIP("192.168.1." + string(rune(i%256))),
		}
	}
	r.Update(devices)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.GetDevices()
	}
}

func BenchmarkRegistryCount(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := make([]scanner.Device, 50)
	for i := 0; i < 50; i++ {
		devices[i] = scanner.Device{IP: net.ParseIP("192.168.1.1")}
	}
	r.Update(devices)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Count()
	}
}

func BenchmarkRegistryOnlineCount(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := make([]scanner.Device, 50)
	for i := 0; i < 50; i++ {
		devices[i] = scanner.Device{IP: net.ParseIP("192.168.1.1")}
	}
	r.Update(devices)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.OnlineCount()
	}
}

func BenchmarkRegistryExport(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	devices := make([]scanner.Device, 20)
	for i := 0; i < 20; i++ {
		devices[i] = scanner.Device{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "test-host",
			Ports:    []int{80, 443},
		}
	}
	r.Update(devices)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Export()
	}
}

func BenchmarkStorageSaveDevice(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")
	storage, _ := NewStorage(dbPath, 24*time.Hour)
	defer storage.Close()

	device := scanner.Device{
		IP:       net.ParseIP("192.168.1.1"),
		MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Hostname: "bench-device",
		Metadata: make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SaveDevice(device)
	}
}

func BenchmarkStorageLoadDevices(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")
	storage, _ := NewStorage(dbPath, 24*time.Hour)
	defer storage.Close()

	// Pre-populate with devices
	for i := 0; i < 100; i++ {
		device := scanner.Device{
			IP:       net.ParseIP("192.168.1." + string(rune(i%256))),
			Metadata: make(map[string]interface{}),
		}
		storage.SaveDevice(device)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.LoadDevices()
	}
}

func BenchmarkAlertGeneration(b *testing.B) {
	r := NewRegistry(5 * time.Minute)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		devices := []scanner.Device{
			{IP: net.ParseIP("192.168.1." + string(rune(i%256)))},
		}
		r.Update(devices)
		r.GetAlerts() // Drain alerts
	}
}

func TestRegistryGetDevices(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Hostname: "host1"},
		{IP: net.ParseIP("192.168.1.2"), Hostname: "host2"},
		{IP: net.ParseIP("192.168.1.3"), Hostname: "host3"},
	}
	r.Update(devices)

	result := r.GetDevices()
	if len(result) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(result))
	}
}

func TestRegistryAlertChan(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	ch := r.AlertChan()
	if ch == nil {
		t.Error("Expected non-nil alert channel")
	}
}

func TestRegistryAddRule(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	rule := AlertRule{
		Condition: func(d scanner.Device) bool {
			return d.IP.String() == "192.168.1.1"
		},
		Action: func(a Alert) {
			// Custom action
		},
	}
	r.AddRule(rule)

	// Rule should be added (no error expected)
}

func TestRegistryDetectChangesPortChange(t *testing.T) {
	r := NewRegistry(5 * time.Minute)

	// First update with initial ports
	devices1 := []scanner.Device{
		{
			IP:    net.ParseIP("192.168.1.1"),
			Ports: []int{22},
		},
	}
	r.Update(devices1)
	r.GetAlerts() // Clear new device alert

	// Second update with additional ports
	devices2 := []scanner.Device{
		{
			IP:    net.ParseIP("192.168.1.1"),
			Ports: []int{22, 80, 443},
		},
	}
	r.Update(devices2)

	time.Sleep(10 * time.Millisecond)
	alerts := r.GetAlerts()

	foundPortChange := false
	for _, a := range alerts {
		if a.Type == AlertPortChange {
			foundPortChange = true
			break
		}
	}

	if !foundPortChange {
		t.Error("Expected port change alert")
	}
}

func TestRegistryDetectChangesOffline(t *testing.T) {
	r := NewRegistry(1 * time.Millisecond) // Very short threshold

	// Add device
	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1")},
	}
	r.Update(devices)
	r.GetAlerts() // Clear new device alert

	// Wait past threshold
	time.Sleep(10 * time.Millisecond)

	// Update with empty list
	r.Update([]scanner.Device{})

	time.Sleep(10 * time.Millisecond)
	alerts := r.GetAlerts()

	foundOffline := false
	for _, a := range alerts {
		if a.Type == AlertDeviceOffline {
			foundOffline = true
			break
		}
	}

	if !foundOffline {
		t.Error("Expected offline alert")
	}
}

func TestStorageCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 1*time.Millisecond) // Very short retention
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Save a device
	device := scanner.Device{
		IP:       net.ParseIP("192.168.1.1"),
		Metadata: make(map[string]interface{}),
	}
	storage.SaveDevice(device)

	// Wait for retention period
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	err = storage.Cleanup()
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

func TestNewRegistryWithStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create registry (storage is not a parameter)
	r := NewRegistry(5 * time.Minute)
	if r == nil {
		t.Fatal("Expected non-nil registry")
	}
}
