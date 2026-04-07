// Package script provides the embedded Tengo scripting engine.
package script

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// TengoEngine implements the scripting engine using d5/tengo.
type TengoEngine struct {
	maxTime   time.Duration
	maxAllocs int64
	scripts   map[string]*tengo.Compiled
	mu        sync.RWMutex
	running   bool
	cancel    context.CancelFunc
	api       *APIBridge
}

// NewTengoEngine creates a new Tengo scripting engine.
func NewTengoEngine(maxTime time.Duration, maxMemoryMB int) *TengoEngine {
	// Convert MB to approximate allocation count
	// Rough estimate: 1 allocation ~= 100 bytes on average
	maxAllocs := int64(maxMemoryMB * 1024 * 1024 / 100)
	if maxAllocs < 1000 {
		maxAllocs = 1000
	}

	return &TengoEngine{
		maxTime:   maxTime,
		maxAllocs: maxAllocs,
		scripts:   make(map[string]*tengo.Compiled),
		api:       NewAPIBridge(),
	}
}

// SetAPIBridge sets the API bridge for the engine.
func (e *TengoEngine) SetAPIBridge(api *APIBridge) {
	e.api = api
}

// Run executes a script from a string.
func (e *TengoEngine) Run(ctx context.Context, source string) error {
	script := tengo.NewScript([]byte(source))

	// Add standard library modules
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	// Add custom API functions
	if err := e.addAPIFunctions(script); err != nil {
		return fmt.Errorf("failed to add API functions: %w", err)
	}

	// Set resource limits
	script.SetMaxAllocs(e.maxAllocs)

	// Compile the script
	compiled, err := script.Compile()
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	// Create context with timeout
	runCtx, cancel := context.WithTimeout(ctx, e.maxTime)
	defer cancel()

	e.mu.Lock()
	e.running = true
	e.cancel = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.cancel = nil
		e.mu.Unlock()
	}()

	// Run with context
	errCh := make(chan error, 1)
	go func() {
		errCh <- compiled.Run()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("runtime error: %w", err)
		}
		return nil
	case <-runCtx.Done():
		return fmt.Errorf("script execution timed out after %v", e.maxTime)
	}
}

// LoadFile loads and runs a script from a file.
func (e *TengoEngine) LoadFile(ctx context.Context, path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	return e.Run(ctx, string(source))
}

// Stop stops any running script.
func (e *TengoEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}
}

// IsRunning returns whether a script is currently running.
func (e *TengoEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// addAPIFunctions adds custom API functions to the script.
func (e *TengoEngine) addAPIFunctions(script *tengo.Script) error {
	funcs := []struct {
		name string
		fn   tengo.CallableFunc
	}{
		{"scan", e.apiFuncScan},
		{"ping", e.apiFuncPing},
		{"portScan", e.apiFuncPortScan},
		{"resolve", e.apiFuncResolve},
		{"alert", e.apiFuncAlert},
		{"getDevices", e.apiFuncGetDevices},
		{"get", e.apiFuncGet},
		{"set", e.apiFuncSet},
		{"print", apiFuncPrint},
		{"println", apiFuncPrintln},
	}

	for _, f := range funcs {
		if err := script.Add(f.name, &tengo.UserFunction{Name: f.name, Value: f.fn}); err != nil {
			return err
		}
	}
	return nil
}

// apiFuncScan implements the scan() scripting API function.
func (e *TengoEngine) apiFuncScan(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	subnet, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "subnet", Expected: "string"}
	}
	return tengo.FromInterface(e.api.Scan(subnet))
}

// apiFuncPing implements the ping() scripting API function.
func (e *TengoEngine) apiFuncPing(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	host, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "host", Expected: "string"}
	}
	ok, rtt := e.api.Ping(host)
	return tengo.FromInterface(map[string]interface{}{"ok": ok, "rtt": rtt.Milliseconds()})
}

// apiFuncPortScan implements the portScan() scripting API function.
func (e *TengoEngine) apiFuncPortScan(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	host, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "host", Expected: "string"}
	}
	ports, err := extractPortsArray(args[1])
	if err != nil {
		return nil, err
	}
	return tengo.FromInterface(e.api.PortScan(host, ports))
}

// extractPortsArray extracts an int slice from a tengo Array.
func extractPortsArray(obj tengo.Object) ([]int, error) {
	portsObj, ok := obj.(*tengo.Array)
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "ports", Expected: "array"}
	}
	var ports []int
	for _, p := range portsObj.Value {
		if pi, ok := tengo.ToInt(p); ok {
			ports = append(ports, pi)
		}
	}
	return ports, nil
}

// apiFuncResolve implements the resolve() scripting API function.
func (e *TengoEngine) apiFuncResolve(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	hostname, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "hostname", Expected: "string"}
	}
	return tengo.FromInterface(e.api.Resolve(hostname))
}

// apiFuncAlert implements the alert() scripting API function.
func (e *TengoEngine) apiFuncAlert(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	level, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "level", Expected: "string"}
	}
	message, ok := tengo.ToString(args[1])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "message", Expected: "string"}
	}
	e.api.Alert(level, message)
	return tengo.UndefinedValue, nil
}

// apiFuncGetDevices implements the getDevices() scripting API function.
func (e *TengoEngine) apiFuncGetDevices(args ...tengo.Object) (tengo.Object, error) {
	return tengo.FromInterface(e.api.GetDevices())
}

// apiFuncGet implements the get() scripting API function for key-value storage.
func (e *TengoEngine) apiFuncGet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	key, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "key", Expected: "string"}
	}
	value := e.api.Get(key)
	if value == nil {
		return tengo.UndefinedValue, nil
	}
	return tengo.FromInterface(value)
}

// apiFuncSet implements the set() scripting API function for key-value storage.
func (e *TengoEngine) apiFuncSet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	key, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "key", Expected: "string"}
	}
	e.api.Set(key, tengo.ToInterface(args[1]))
	return tengo.UndefinedValue, nil
}

// apiFuncPrint implements the print() scripting API function.
func apiFuncPrint(args ...tengo.Object) (tengo.Object, error) {
	for _, arg := range args {
		fmt.Print(tengo.ToInterface(arg))
	}
	return tengo.UndefinedValue, nil
}

// apiFuncPrintln implements the println() scripting API function.
func apiFuncPrintln(args ...tengo.Object) (tengo.Object, error) {
	for _, arg := range args {
		fmt.Print(tengo.ToInterface(arg))
	}
	fmt.Println()
	return tengo.UndefinedValue, nil
}
