package tui

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/tuimap/internal/scanner"
	"github.com/opd-ai/tuimap/internal/tracker"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.currentView != ViewDeviceList {
		t.Errorf("Expected default view to be ViewDeviceList, got %d", m.currentView)
	}

	if len(m.devices) != 0 {
		t.Errorf("Expected empty devices slice, got %d", len(m.devices))
	}

	if m.status != "Ready - Press 's' to scan" {
		t.Errorf("Unexpected initial status: %s", m.status)
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("Expected nil command from Init")
	}
}

func TestModelUpdate(t *testing.T) {
	m := NewModel()
	m.ready = true

	tests := []struct {
		name         string
		msg          tea.Msg
		expectedView ViewType
	}{
		{"switch to network map", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}, ViewNetworkMap},
		{"switch to device list", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}, ViewDeviceList},
		{"switch to tool view", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}, ViewToolView},
		{"switch to script console", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}, ViewScriptConsole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := m.Update(tt.msg)
			updated := newModel.(Model)
			if updated.currentView != tt.expectedView {
				t.Errorf("Expected view %d, got %d", tt.expectedView, updated.currentView)
			}
		})
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel()

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.width != 100 {
		t.Errorf("Expected width 100, got %d", updated.width)
	}

	if updated.height != 40 {
		t.Errorf("Expected height 40, got %d", updated.height)
	}

	if !updated.ready {
		t.Error("Expected ready to be true after window size message")
	}
}

func TestModelView(t *testing.T) {
	m := NewModel()

	// Not ready
	view := m.View()
	if view != "Loading..." {
		t.Errorf("Expected 'Loading...', got '%s'", view)
	}

	// Ready
	m.ready = true
	m.width = 80
	m.height = 24

	view = m.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}
}

func TestSetDevices(t *testing.T) {
	m := NewModel()

	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Hostname: "test-host",
			Status:   scanner.StatusOnline,
		},
	}

	m.SetDevices(devices)

	if len(m.devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(m.devices))
	}

	if m.lastUpdate.IsZero() {
		t.Error("Expected lastUpdate to be set")
	}
}

func TestSetAlerts(t *testing.T) {
	m := NewModel()

	alerts := []tracker.Alert{
		{
			Type:      tracker.AlertNewDevice,
			Timestamp: time.Now(),
			Message:   "Test alert",
		},
	}

	m.SetAlerts(alerts)

	if len(m.alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(m.alerts))
	}
}

func TestSetStatus(t *testing.T) {
	m := NewModel()

	m.SetStatus("Testing status")

	if m.status != "Testing status" {
		t.Errorf("Expected 'Testing status', got '%s'", m.status)
	}
}

func TestNewStyles(t *testing.T) {
	styles := NewStyles()

	// Just verify styles are created without panic
	_ = styles.Title.Render("test")
	_ = styles.Subtitle.Render("test")
	_ = styles.Status.Render("test")
	_ = styles.Active.Render("test")
	_ = styles.Inactive.Render("test")
	_ = styles.Online.Render("test")
	_ = styles.Offline.Render("test")
	_ = styles.New.Render("test")
	_ = styles.Border.Render("test")
	_ = styles.HelpStyle.Render("test")
}

func TestViewTypes(t *testing.T) {
	// Verify view type constants
	if ViewNetworkMap != 0 {
		t.Errorf("Expected ViewNetworkMap to be 0, got %d", ViewNetworkMap)
	}
	if ViewDeviceList != 1 {
		t.Errorf("Expected ViewDeviceList to be 1, got %d", ViewDeviceList)
	}
	if ViewToolView != 2 {
		t.Errorf("Expected ViewToolView to be 2, got %d", ViewToolView)
	}
	if ViewScriptConsole != 3 {
		t.Errorf("Expected ViewScriptConsole to be 3, got %d", ViewScriptConsole)
	}
}

func TestRenderTabs(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	tabs := m.renderTabs()
	if tabs == "" {
		t.Error("Expected non-empty tabs")
	}
}

func TestRenderStatusBar(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	status := m.renderStatusBar()
	if status == "" {
		t.Error("Expected non-empty status bar")
	}
}

func TestRenderHelp(t *testing.T) {
	m := NewModel()

	help := m.renderHelp()
	if help == "" {
		t.Error("Expected non-empty help text")
	}
}

func TestUpdateTableRows(t *testing.T) {
	m := NewModel()
	m.devices = []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1"),
			MAC:      net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Hostname: "test-host",
			Vendor:   "Test Vendor",
			Status:   scanner.StatusOnline,
			Ports:    []int{80, 443},
		},
	}

	// Should not panic
	m.updateTableRows()
}

func TestRenderNetworkMapEmpty(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	result := m.renderNetworkMap()
	if result == "" {
		t.Error("Expected non-empty network map for empty devices")
	}
}

func TestRenderNetworkMapWithDevices(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.devices = []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.2"), Status: scanner.StatusOffline},
		{IP: net.ParseIP("192.168.1.3"), Status: scanner.StatusNew},
	}

	result := m.renderNetworkMap()
	if result == "" {
		t.Error("Expected non-empty network map")
	}
}

func TestRenderToolView(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	result := m.renderToolView()
	if result == "" {
		t.Error("Expected non-empty tool view")
	}
}

func TestRenderScriptConsole(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	result := m.renderScriptConsole()
	if result == "" {
		t.Error("Expected non-empty script console")
	}
}

func TestToolViewInteraction(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView

	// Test tool selection with key 5 (first tool: netcat)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selectedTool != 0 {
		t.Errorf("Expected selectedTool 0, got %d", updated.selectedTool)
	}
	if !updated.toolFocused {
		t.Error("Expected toolFocused to be true after selecting tool")
	}
}

func TestToolViewRenderWithSelection(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView
	m.selectedTool = 0

	result := m.renderToolView()
	if result == "" {
		t.Error("Expected non-empty tool view")
	}
	// Check that selection indicator is present
	if !strings.Contains(result, "netcat") {
		t.Error("Expected tool view to contain 'netcat'")
	}
}

func TestToolViewEscapeCancelsInput(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView
	m.selectedTool = 0
	m.toolFocused = true
	m.toolInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.toolFocused {
		t.Error("Expected toolFocused to be false after Esc")
	}
}

func TestToolsInitialized(t *testing.T) {
	m := NewModel()

	if len(m.tools) != 5 {
		t.Errorf("Expected 5 tools, got %d", len(m.tools))
	}

	expectedNames := []string{"netcat", "telnet", "traceroute", "dig", "whois"}
	for i, name := range expectedNames {
		if m.tools[i].Name() != name {
			t.Errorf("Expected tool %d to be %s, got %s", i, name, m.tools[i].Name())
		}
	}
}

func TestToolResultMsg(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.toolRunning = true

	msg := toolResultMsg{output: "test output\n", done: true}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.toolRunning {
		t.Error("Expected toolRunning to be false after done message")
	}
	if !strings.Contains(updated.toolOutputText, "test output") {
		t.Error("Expected toolOutputText to contain 'test output'")
	}
}

func TestScriptConsoleInitialized(t *testing.T) {
	m := NewModel()

	if m.scriptEngine == nil {
		t.Error("Expected scriptEngine to be initialized")
	}
	if m.scriptsDir == "" {
		t.Error("Expected scriptsDir to be set")
	}
}

func TestScriptConsoleView(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole

	result := m.renderScriptConsole()
	if result == "" {
		t.Error("Expected non-empty script console")
	}
	if !strings.Contains(result, "Script Console") {
		t.Error("Expected script console header")
	}
}

func TestScriptConsoleFocus(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	// Switch to script console
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.currentView != ViewScriptConsole {
		t.Errorf("Expected ViewScriptConsole, got %d", updated.currentView)
	}
	if !updated.scriptFocused {
		t.Error("Expected scriptFocused to be true")
	}
}

func TestScriptConsoleEscape(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptFocused = true
	m.scriptInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.scriptFocused {
		t.Error("Expected scriptFocused to be false after Esc")
	}
}

func TestScriptResultMsg(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.scriptRunning = true

	msg := scriptResultMsg{output: "", err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.scriptRunning {
		t.Error("Expected scriptRunning to be false after result message")
	}
}

func TestExecuteScriptCommandEmpty(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue("   ")

	newModel, cmd := m.executeScriptCommand()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command for empty input")
	}
	if updated.scriptInput.Value() != "" {
		t.Error("Expected input to be cleared")
	}
}

func TestExecuteScriptCommandHelp(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue(":help")

	newModel, cmd := m.executeScriptCommand()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command for :help")
	}
	if !strings.Contains(updated.scriptOutputText, "Commands:") {
		t.Error("Expected help text in output")
	}
}

func TestExecuteScriptCommandStop(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue(":stop")

	newModel, _ := m.executeScriptCommand()
	updated := newModel.(Model)

	if !strings.Contains(updated.scriptOutputText, "No script is running") {
		t.Error("Expected 'No script is running' message")
	}
}

func TestExecuteScriptCommandStopRunning(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptRunning = true
	m.scriptInput.SetValue(":stop")

	newModel, _ := m.executeScriptCommand()
	updated := newModel.(Model)

	if !strings.Contains(updated.scriptOutputText, "Script stopped") {
		t.Error("Expected 'Script stopped' message")
	}
}

func TestExecuteScriptCommandUnknown(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue(":unknown")

	newModel, _ := m.executeScriptCommand()
	updated := newModel.(Model)

	if !strings.Contains(updated.scriptOutputText, "Unknown command") {
		t.Error("Expected 'Unknown command' message")
	}
}

func TestExecuteScriptCommandLoadMissingArg(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue(":load")

	newModel, _ := m.executeScriptCommand()
	updated := newModel.(Model)

	if !strings.Contains(updated.scriptOutputText, "Usage:") {
		t.Error("Expected usage message for :load without argument")
	}
}

func TestExecuteScriptCommandInlineScript(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewScriptConsole
	m.scriptInput.SetValue("fmt := import(\"fmt\"); fmt.println(\"test\")")

	newModel, cmd := m.executeScriptCommand()
	updated := newModel.(Model)

	if cmd == nil {
		t.Error("Expected non-nil command for inline script")
	}
	if !updated.scriptRunning {
		t.Error("Expected scriptRunning to be true")
	}
}

func TestExecuteSelectedToolNoSelection(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView
	m.selectedTool = -1

	newModel, cmd := m.executeSelectedTool()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command when no tool selected")
	}
	if updated.status != "No tool selected" {
		t.Errorf("Expected 'No tool selected' status, got '%s'", updated.status)
	}
}

func TestExecuteSelectedToolInvalidArgs(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView
	m.selectedTool = 0 // netcat
	m.toolInput.SetValue("") // Empty args - should fail validation

	newModel, cmd := m.executeSelectedTool()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command for invalid args")
	}
	if !strings.Contains(updated.toolOutputText, "Error:") {
		t.Error("Expected error message in tool output")
	}
}

func TestExecuteSelectedToolValidArgs(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.currentView = ViewToolView
	m.selectedTool = 3 // dig
	m.toolInput.SetValue("example.com A")

	newModel, cmd := m.executeSelectedTool()
	updated := newModel.(Model)

	if cmd == nil {
		t.Error("Expected non-nil command for valid tool execution")
	}
	if !updated.toolRunning {
		t.Error("Expected toolRunning to be true")
	}
	if !strings.Contains(updated.toolOutputText, "dig") {
		t.Error("Expected tool name in output")
	}
}

func TestListScriptsNonExistentDir(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.scriptsDir = "/nonexistent/path/to/scripts"

	newModel, cmd := m.listScripts()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command for non-existent directory")
	}
	if !strings.Contains(updated.scriptOutputText, "Cannot read scripts directory") {
		t.Error("Expected error message about directory")
	}
}

func TestLoadScriptNotFound(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.scriptsDir = "/tmp" // Exists but no script files

	newModel, cmd := m.loadScript("nonexistent.tengo")
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("Expected nil command for non-existent script")
	}
	if !strings.Contains(updated.scriptOutputText, "Script not found") {
		t.Error("Expected 'Script not found' message")
	}
}

func TestScriptResultMsgWithError(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.scriptRunning = true

	msg := scriptResultMsg{output: "", err: fmt.Errorf("test error")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.scriptRunning {
		t.Error("Expected scriptRunning to be false after error")
	}
	if !strings.Contains(updated.scriptOutputText, "Error:") {
		t.Error("Expected error message in output")
	}
}

func TestToolResultMsgPartial(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.toolRunning = true

	// Test partial result (not done)
	msg := toolResultMsg{output: "partial output\n", done: false}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if !updated.toolRunning {
		t.Error("Expected toolRunning to remain true for partial result")
	}
	if !strings.Contains(updated.toolOutputText, "partial output") {
		t.Error("Expected partial output in tool output")
	}
}

func TestScanResultMsgSuccess(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.status = "Scanning..."

	result := &scanner.ScanResult{
		Devices: []scanner.Device{
			{IP: net.ParseIP("192.168.1.1"), Status: scanner.StatusOnline},
		},
	}
	msg := scanResultMsg{result: result, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(updated.devices))
	}
	if !strings.Contains(updated.status, "Scan complete") {
		t.Errorf("Expected 'Scan complete' in status, got '%s'", updated.status)
	}
}

func TestScanResultMsgError(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.status = "Scanning..."

	msg := scanResultMsg{result: nil, err: fmt.Errorf("scan failed")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if !strings.Contains(updated.status, "Scan error") {
		t.Errorf("Expected 'Scan error' in status, got '%s'", updated.status)
	}
}

func TestRenderDeviceList(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.devices = []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), MAC: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}, Status: scanner.StatusOnline},
	}
	m.updateTableRows()

	result := m.renderDeviceList()
	if result == "" {
		t.Error("Expected non-empty device list")
	}
}
