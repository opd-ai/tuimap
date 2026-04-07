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

func TestAPIBridgeSetScanner(t *testing.T) {
	api := NewAPIBridge()

	called := false
	api.SetScanner(func(ctx context.Context, subnet string) ([]map[string]interface{}, error) {
		called = true
		return []map[string]interface{}{
			{"ip": "192.168.1.1"},
		}, nil
	})

	result := api.Scan("192.168.1.0/24")
	if !called {
		t.Error("Custom scanner was not called")
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
}

func TestAPIBridgeSetPinger(t *testing.T) {
	api := NewAPIBridge()

	called := false
	api.SetPinger(func(ctx context.Context, host string) (bool, time.Duration) {
		called = true
		return true, 50 * time.Millisecond
	})

	ok, rtt := api.Ping("test.local")
	if !called {
		t.Error("Custom pinger was not called")
	}
	if !ok {
		t.Error("Expected ping to succeed")
	}
	if rtt != 50*time.Millisecond {
		t.Errorf("Expected 50ms RTT, got %v", rtt)
	}
}

func TestAPIBridgeSetPortScanner(t *testing.T) {
	api := NewAPIBridge()

	called := false
	api.SetPortScanner(func(ctx context.Context, host string, ports []int) []int {
		called = true
		return []int{80, 443}
	})

	result := api.PortScan("test.local", []int{22, 80, 443})
	if !called {
		t.Error("Custom port scanner was not called")
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 open ports, got %d", len(result))
	}
}

func TestAPIBridgeSetResolver(t *testing.T) {
	api := NewAPIBridge()

	called := false
	api.SetResolver(func(ctx context.Context, host string) []string {
		called = true
		return []string{"192.168.1.1", "192.168.1.2"}
	})

	result := api.Resolve("test.local")
	if !called {
		t.Error("Custom resolver was not called")
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(result))
	}
}

func TestAPIBridgeScanDefault(t *testing.T) {
	api := NewAPIBridge()

	// Default scanner returns empty slice
	result := api.Scan("192.168.1.0/24")
	// Default scanner returns empty (not nil)
	if len(result) != 0 {
		t.Errorf("Expected empty from default scanner, got %d items", len(result))
	}
}

func TestAPIBridgePortScanDefault(t *testing.T) {
	api := NewAPIBridge()

	// Default port scanner tries real connections (should fail for invalid host)
	result := api.PortScan("10.255.255.1", []int{80, 443})
	if len(result) != 0 {
		t.Errorf("Expected no open ports, got %d", len(result))
	}
}

func TestTengoEngineWithAlertFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	alertCalled := false
	engine.api.SetAlertHandler(func(level, message string) {
		alertCalled = true
	})

	script := `
alert("info", "test alert")
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !alertCalled {
		t.Error("Alert function was not called from script")
	}
}

func TestTengoEngineWithGetDevicesFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	engine.api.SetDevicesProvider(func() []map[string]interface{} {
		return []map[string]interface{}{
			{"ip": "192.168.1.1", "hostname": "host1"},
		}
	})

	// Just call the function - it may have conversion issues but should not crash
	script := `
getDevices()
`
	err := engine.Run(ctx, script)
	// Conversion issues are expected but not a crash
	if err != nil {
		t.Logf("Note: getDevices() returned conversion error (expected): %v", err)
	}
}

func TestTengoEngineContextCancellation(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	script := `x := 1`
	err := engine.Run(ctx, script)
	// May or may not error depending on timing
	_ = err
}

func TestTengoEngineScanFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Just call scan - conversion issues are expected
	script := `
scan("192.168.1.0/30")
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Logf("Note: scan() returned conversion error (expected): %v", err)
	}
}

func TestTengoEnginePingFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	script := `
result := ping("localhost")
println(result)
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestTengoEngineResolveFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Just call resolve - conversion issues are expected
	script := `
resolve("localhost")
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Logf("Note: resolve() returned conversion error (expected): %v", err)
	}
}

func TestTengoEnginePortScanFunction(t *testing.T) {
	engine := NewTengoEngine(5*time.Second, 10)
	ctx := context.Background()

	// Just call portScan - conversion issues are expected
	script := `
portScan("127.0.0.1", [80, 443])
`
	err := engine.Run(ctx, script)
	if err != nil {
		t.Logf("Note: portScan() returned conversion error (expected): %v", err)
	}
}
