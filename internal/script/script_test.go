package script

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTengoEngine(t *testing.T) {
	engine := NewTengoEngine(30*time.Second, 50)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	if engine.maxTime != 30*time.Second {
		t.Errorf("Expected maxTime 30s, got %v", engine.maxTime)
	}

	if engine.api == nil {
		t.Error("Expected non-nil API bridge")
	}
}

func TestTengoEngineRun(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Simple script that should succeed
	script := `
x := 1 + 2
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestTengoEngineRunWithPrint(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Script using println
	script := `
println("Hello, TuiMap!")
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestTengoEngineRunCompileError(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Invalid script
	script := `
this is not valid tengo
`
	err := engine.Run(ctx, script)
	if err == nil {
		t.Error("Expected compilation error")
	}
}

func TestTengoEngineRunTimeout(t *testing.T) {
	engine := NewTengoEngine(100*time.Millisecond, 10)
	ctx := context.Background()

	// Script that would run forever
	script := `
for i := 0; i < 1000000000; i++ {
    x := i * 2
}
`
	err := engine.Run(ctx, script)
	// Should either timeout or hit allocation limit
	if err == nil {
		t.Error("Expected timeout or resource limit error")
	}
}

func TestTengoEngineLoadFile(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Create temp script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.tengo")
	content := `x := 1 + 1`
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	err := engine.LoadFile(ctx, scriptPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestTengoEngineLoadFileNotFound(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	err := engine.LoadFile(ctx, "/nonexistent/path/script.tengo")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestTengoEngineStop(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)

	// Should not panic when nothing is running
	engine.Stop()

	if engine.IsRunning() {
		t.Error("Expected not running after stop")
	}
}

func TestTengoEngineIsRunning(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)

	if engine.IsRunning() {
		t.Error("Expected not running initially")
	}
}

func TestAPIBridgeNew(t *testing.T) {
	api := NewAPIBridge()

	if api == nil {
		t.Fatal("Expected non-nil API bridge")
	}

	if api.storage == nil {
		t.Error("Expected non-nil storage")
	}
}

func TestAPIBridgeGetSet(t *testing.T) {
	api := NewAPIBridge()

	// Test set and get
	api.Set("key1", "value1")
	val := api.Get("key1")
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test get non-existent
	val = api.Get("nonexistent")
	if val != nil {
		t.Errorf("Expected nil, got %v", val)
	}
}

func TestAPIBridgeDelete(t *testing.T) {
	api := NewAPIBridge()

	api.Set("key1", "value1")
	api.Delete("key1")
	val := api.Get("key1")
	if val != nil {
		t.Errorf("Expected nil after delete, got %v", val)
	}
}

func TestAPIBridgeResolve(t *testing.T) {
	api := NewAPIBridge()

	// Try to resolve localhost
	ips := api.Resolve("localhost")
	if len(ips) == 0 {
		t.Skip("localhost resolution not available")
	}
}

func TestAPIBridgeAlert(t *testing.T) {
	api := NewAPIBridge()

	called := false
	api.SetAlertHandler(func(level, message string) {
		called = true
		if level != "info" {
			t.Errorf("Expected 'info' level, got '%s'", level)
		}
		if message != "test message" {
			t.Errorf("Expected 'test message', got '%s'", message)
		}
	})

	api.Alert("info", "test message")

	if !called {
		t.Error("Alert handler was not called")
	}
}

func TestAPIBridgeGetDevices(t *testing.T) {
	api := NewAPIBridge()

	// Default provider returns empty
	devices := api.GetDevices()
	if len(devices) != 0 {
		t.Errorf("Expected empty devices, got %d", len(devices))
	}

	// Test custom provider
	api.SetDevicesProvider(func() []map[string]interface{} {
		return []map[string]interface{}{
			{"ip": "192.168.1.1", "hostname": "test"},
		}
	})

	devices = api.GetDevices()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}
}

func TestAPIBridgePing(t *testing.T) {
	api := NewAPIBridge()

	// Default pinger tries TCP ports
	ok, rtt := api.Ping("nonexistent.invalid.local")
	if ok {
		t.Error("Expected ping to fail for invalid host")
	}
	_ = rtt // RTT is 0 on failure
}

func TestTengoEngineAPIFunctions(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Test using API functions in script
	script := `
// Use get/set functions
set("test_key", "test_value")
value := get("test_key")

// Use println
println("Test completed")
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify storage was updated
	value := engine.api.Get("test_key")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %v", value)
	}
}

func TestTengoEngineSetAPIBridge(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	newAPI := NewAPIBridge()

	engine.SetAPIBridge(newAPI)

	if engine.api != newAPI {
		t.Error("API bridge was not updated")
	}
}

func TestDefaultPortScanner(t *testing.T) {
	ctx := context.Background()

	// Should return empty for non-responsive host
	ports := defaultPortScanner(ctx, "10.255.255.1", []int{80, 443})
	if len(ports) != 0 {
		t.Errorf("Expected no open ports, got %d", len(ports))
	}
}

func TestDefaultResolver(t *testing.T) {
	ctx := context.Background()

	// Should fail for invalid hostname
	ips := defaultResolver(ctx, "nonexistent.invalid.tld")
	if len(ips) != 0 {
		t.Errorf("Expected no IPs for invalid hostname, got %d", len(ips))
	}
}
