// Package tui provides the Bubble Tea terminal user interface.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opd-ai/tuimap/internal/scanner"
	"github.com/opd-ai/tuimap/internal/tracker"
)

// ViewType represents the current view
type ViewType int

const (
	ViewNetworkMap ViewType = iota
	ViewDeviceList
	ViewToolView
	ViewScriptConsole
)

// scanResultMsg is sent when a scan completes.
type scanResultMsg struct {
	result *scanner.ScanResult
	err    error
}

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	width        int
	height       int
	currentView  ViewType
	devices      []scanner.Device
	alerts       []tracker.Alert
	table        table.Model
	styles       Styles
	ready        bool
	scanResult   *scanner.ScanResult
	status       string
	lastUpdate   time.Time
	orchestrator *scanner.Orchestrator
	subnet       string
	scanning     bool
}

// Styles holds the lipgloss styles for the TUI.
type Styles struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Status    lipgloss.Style
	Active    lipgloss.Style
	Inactive  lipgloss.Style
	Online    lipgloss.Style
	Offline   lipgloss.Style
	New       lipgloss.Style
	Border    lipgloss.Style
	HelpStyle lipgloss.Style
}

// NewStyles creates the default styles.
func NewStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginLeft(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(1),
		Active: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		Inactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Online: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")),
		Offline: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		New: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")),
		HelpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// NewModel creates a new TUI model.
func NewModel() Model {
	return NewModelWithOrchestrator(nil, "")
}

// NewModelWithOrchestrator creates a new TUI model with a scanner orchestrator.
func NewModelWithOrchestrator(orch *scanner.Orchestrator, subnet string) Model {
	columns := []table.Column{
		{Title: "IP", Width: 15},
		{Title: "MAC", Width: 17},
		{Title: "Hostname", Width: 20},
		{Title: "Vendor", Width: 15},
		{Title: "Status", Width: 10},
		{Title: "Ports", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return Model{
		currentView:  ViewDeviceList,
		table:        t,
		styles:       NewStyles(),
		devices:      make([]scanner.Device, 0),
		alerts:       make([]tracker.Alert, 0),
		status:       "Ready - Press 's' to scan",
		orchestrator: orch,
		subnet:       subnet,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = ViewNetworkMap
			m.status = "Network Map View"
		case "2":
			m.currentView = ViewDeviceList
			m.status = "Device List View"
		case "3":
			m.currentView = ViewToolView
			m.status = "Tool View"
		case "4":
			m.currentView = ViewScriptConsole
			m.status = "Script Console View"
		case "s":
			if !m.scanning && m.orchestrator != nil && m.subnet != "" {
				m.scanning = true
				m.status = "Scanning..."
				return m, m.startScan()
			} else if m.orchestrator == nil {
				m.status = "Scanner not configured"
			} else if m.subnet == "" {
				m.status = "No subnet configured"
			} else {
				m.status = "Scan already in progress..."
			}
		case "r":
			m.status = "Refreshing..."
		}

		if m.currentView == ViewDeviceList {
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case scanResultMsg:
		m.scanning = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Scan error: %v", msg.err)
		} else {
			m.devices = msg.result.Devices
			m.scanResult = msg.result
			m.lastUpdate = time.Now()
			m.status = fmt.Sprintf("Scan complete: %d devices found in %v", len(m.devices), msg.result.ScanTime.Round(time.Millisecond))
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(m.width - 4)
		m.table.SetHeight(m.height - 10)
		m.ready = true
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var builder strings.Builder

	// Header
	title := m.styles.Title.Render("TuiMap")
	subtitle := m.styles.Subtitle.Render(" - Network Analysis Tool")
	builder.WriteString(title + subtitle + "\n\n")

	// Tab bar
	builder.WriteString(m.renderTabs() + "\n\n")

	// Main content
	switch m.currentView {
	case ViewNetworkMap:
		builder.WriteString(m.renderNetworkMap())
	case ViewDeviceList:
		builder.WriteString(m.renderDeviceList())
	case ViewToolView:
		builder.WriteString(m.renderToolView())
	case ViewScriptConsole:
		builder.WriteString(m.renderScriptConsole())
	}

	// Status bar
	builder.WriteString("\n" + m.renderStatusBar())

	// Help
	builder.WriteString("\n" + m.renderHelp())

	return builder.String()
}

// renderTabs renders the tab bar.
func (m Model) renderTabs() string {
	tabs := []string{"[1] Network Map", "[2] Devices", "[3] Tools", "[4] Scripts"}
	var rendered []string

	for i, tab := range tabs {
		if ViewType(i) == m.currentView {
			rendered = append(rendered, m.styles.Active.Render(tab))
		} else {
			rendered = append(rendered, m.styles.Inactive.Render(tab))
		}
	}

	return strings.Join(rendered, " | ")
}

// renderNetworkMap renders the network map view.
func (m Model) renderNetworkMap() string {
	if len(m.devices) == 0 {
		return m.styles.Border.Render("No devices discovered. Press 's' to scan.")
	}

	var builder strings.Builder
	builder.WriteString("Network Topology:\n\n")

	// Simple ASCII representation
	builder.WriteString("  [Gateway]\n")
	builder.WriteString("      │\n")

	for i, device := range m.devices {
		status := "●"
		switch device.Status {
		case scanner.StatusOnline:
			status = m.styles.Online.Render("●")
		case scanner.StatusOffline:
			status = m.styles.Offline.Render("○")
		case scanner.StatusNew:
			status = m.styles.New.Render("★")
		}

		conn := "├──"
		if i == len(m.devices)-1 {
			conn = "└──"
		}

		builder.WriteString(fmt.Sprintf("      %s %s %s (%s)\n", conn, status, device.IP, device.Hostname))
	}

	return m.styles.Border.Width(m.width - 4).Render(builder.String())
}

// renderDeviceList renders the device list view.
func (m Model) renderDeviceList() string {
	m.updateTableRows()
	return m.styles.Border.Width(m.width - 4).Render(m.table.View())
}

// renderToolView renders the tool view.
func (m Model) renderToolView() string {
	tools := []string{"netcat", "telnet", "traceroute", "dig", "whois"}

	var builder strings.Builder
	builder.WriteString("Available Network Tools:\n\n")

	for i, tool := range tools {
		builder.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, tool))
	}

	builder.WriteString("\nPress number to select tool, then enter command arguments.")

	return m.styles.Border.Width(m.width - 4).Render(builder.String())
}

// renderScriptConsole renders the script console view.
func (m Model) renderScriptConsole() string {
	var builder strings.Builder
	builder.WriteString("Script Console (Tengo):\n\n")
	builder.WriteString("Scripts directory: ~/.config/tuimap/scripts\n\n")
	builder.WriteString("Available commands:\n")
	builder.WriteString("  :load <script.tengo>  - Load and run a script\n")
	builder.WriteString("  :list                 - List available scripts\n")
	builder.WriteString("  :stop                 - Stop running script\n\n")
	builder.WriteString("> _")

	return m.styles.Border.Width(m.width - 4).Render(builder.String())
}

// renderStatusBar renders the status bar.
func (m Model) renderStatusBar() string {
	deviceCount := fmt.Sprintf("Devices: %d", len(m.devices))
	alertCount := fmt.Sprintf("Alerts: %d", len(m.alerts))
	lastUpdate := "Last scan: never"
	if !m.lastUpdate.IsZero() {
		lastUpdate = fmt.Sprintf("Last scan: %s", m.lastUpdate.Format("15:04:05"))
	}

	return m.styles.Status.Render(
		fmt.Sprintf("%s | %s | %s | %s", m.status, deviceCount, alertCount, lastUpdate),
	)
}

// renderHelp renders the help text.
func (m Model) renderHelp() string {
	return m.styles.HelpStyle.Render(
		"q: quit | s: scan | r: refresh | 1-4: switch views | ↑↓: navigate",
	)
}

// updateTableRows updates the table with current device data.
func (m *Model) updateTableRows() {
	rows := make([]table.Row, len(m.devices))
	for i, device := range m.devices {
		mac := ""
		if device.MAC != nil {
			mac = device.MAC.String()
		}

		ports := ""
		if len(device.Ports) > 0 {
			portStrs := make([]string, len(device.Ports))
			for j, p := range device.Ports {
				portStrs[j] = fmt.Sprintf("%d", p)
			}
			ports = strings.Join(portStrs, ",")
		}

		rows[i] = table.Row{
			device.IP.String(),
			mac,
			device.Hostname,
			device.Vendor,
			string(device.Status),
			ports,
		}
	}
	m.table.SetRows(rows)
}

// SetDevices updates the device list.
func (m *Model) SetDevices(devices []scanner.Device) {
	m.devices = devices
	m.lastUpdate = time.Now()
}

// SetAlerts updates the alert list.
func (m *Model) SetAlerts(alerts []tracker.Alert) {
	m.alerts = alerts
}

// SetStatus updates the status message.
func (m *Model) SetStatus(status string) {
	m.status = status
}

// startScan returns a command that performs the network scan asynchronously.
func (m Model) startScan() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result, err := m.orchestrator.Scan(ctx, m.subnet)
		return scanResultMsg{result: result, err: err}
	}
}

// Run starts the TUI.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithOrchestrator starts the TUI with a scanner orchestrator.
func RunWithOrchestrator(orch *scanner.Orchestrator, subnet string) error {
	p := tea.NewProgram(NewModelWithOrchestrator(orch, subnet), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
