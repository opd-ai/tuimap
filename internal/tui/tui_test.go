package tui

import (
	"net"
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
