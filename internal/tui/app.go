// Package tui provides the Bubble Tea terminal user interface.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opd-ai/tuimap/internal/scanner"
	"github.com/opd-ai/tuimap/internal/script"
	"github.com/opd-ai/tuimap/internal/tools"
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

// toolResultMsg is sent when tool execution produces output.
type toolResultMsg struct {
	output string
	done   bool
}

// scriptResultMsg is sent when script execution produces output.
type scriptResultMsg struct {
	output string
	err    error
}

// subnetDiscoverMsg is sent when subnet discovery completes.
type subnetDiscoverMsg struct {
	subnets []scanner.SubnetInfo
	err     error
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
	storage      *tracker.Storage
	registry     *tracker.Registry
	subnets      []scanner.SubnetInfo // Discovered subnets
	subnetIdx    int                  // Currently selected subnet index

	// Tool View state
	selectedTool   int
	toolInput      textinput.Model
	toolOutput     viewport.Model
	toolOutputText string
	toolRunning    bool
	tools          []tools.NetworkTool
	toolFocused    bool // true when text input is focused

	// Script Console state
	scriptEngine     *script.TengoEngine
	scriptInput      textinput.Model
	scriptOutput     viewport.Model
	scriptOutputText string
	scriptRunning    bool
	scriptFocused    bool
	scriptsDir       string
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
	return NewModelWithOrchestratorAndStorage(orch, subnet, nil)
}

// NewModelWithOrchestratorAndStorage creates a new TUI model with scanner and storage.
func NewModelWithOrchestratorAndStorage(orch *scanner.Orchestrator, subnet string, storage *tracker.Storage) Model {
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

	// Initialize text inputs and viewports
	ti, vp := newInputAndViewport("Enter arguments...", "Tool output will appear here.")
	si, svp := newInputAndViewport(":load <script>, :list, :stop", "Script output will appear here.")

	// Initialize network tools
	timeout := 10 * time.Second
	networkTools := []tools.NetworkTool{
		tools.NewNetcatTool(timeout),
		tools.NewTelnetTool(timeout),
		tools.NewTracerouteTool(30, timeout),
		tools.NewDigTool(timeout, ""),
		tools.NewWhoisTool(timeout),
	}

	// Initialize script engine
	engine := script.NewTengoEngine(30*time.Second, 50)

	// Get scripts directory
	homeDir, _ := os.UserHomeDir()
	scriptsDir := filepath.Join(homeDir, ".config", "tuimap", "scripts")

	// Create device registry with 5 minute offline threshold
	registry := tracker.NewRegistry(5 * time.Minute)

	// Wire script engine API bridge to real scanner and registry
	apiBridge := script.NewAPIBridge()
	if orch != nil {
		apiBridge.SetScanner(func(ctx context.Context, sub string) ([]map[string]interface{}, error) {
			result, err := orch.Scan(ctx, sub)
			if err != nil {
				return nil, err
			}
			return devicesToMaps(result.Devices), nil
		})
	}
	apiBridge.SetDevicesProvider(func() []map[string]interface{} {
		return devicesToMaps(registry.GetDevices())
	})
	engine.SetAPIBridge(apiBridge)

	// Load previously saved devices from storage
	initialDevices := make([]scanner.Device, 0)
	initialAlerts := make([]tracker.Alert, 0)
	if storage != nil {
		if loaded, err := storage.LoadDevices(); err == nil && len(loaded) > 0 {
			// Preserve persisted device state for the initial model.
			initialDevices = append([]scanner.Device(nil), loaded...)

			// Seed registry with a separate copy because Update mutates
			// device status and timestamps in-place.
			seedDevices := append([]scanner.Device(nil), loaded...)
			_ = registry.Update(seedDevices)
			// Drain any alerts generated from loading
			_ = registry.GetAlerts()
		}
		if loadedAlerts, err := storage.LoadAlerts(); err == nil {
			initialAlerts = loadedAlerts
		}
	}

	return Model{
		currentView:      ViewDeviceList,
		table:            t,
		styles:           NewStyles(),
		devices:          initialDevices,
		alerts:           initialAlerts,
		status:           "Ready - Press 's' to scan",
		orchestrator:     orch,
		subnet:           subnet,
		storage:          storage,
		registry:         registry,
		selectedTool:     -1,
		toolInput:        ti,
		toolOutput:       vp,
		toolOutputText:   "",
		tools:            networkTools,
		toolFocused:      false,
		scriptEngine:     engine,
		scriptInput:      si,
		scriptOutput:     svp,
		scriptOutputText: "",
		scriptFocused:    false,
		scriptsDir:       scriptsDir,
	}
}

// Init initializes the model and discovers available subnets.
func (m Model) Init() tea.Cmd {
	return discoverSubnets
}

// discoverSubnets runs subnet discovery in the background.
func discoverSubnets() tea.Msg {
	subnets, err := scanner.DiscoverSubnets()
	return subnetDiscoverMsg{subnets: subnets, err: err}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case toolResultMsg:
		return m.handleToolResult(msg)
	case scriptResultMsg:
		return m.handleScriptResult(msg)
	case scanResultMsg:
		return m.handleScanResult(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case subnetDiscoverMsg:
		return m.handleSubnetDiscover(msg)
	}
	return m, nil
}

// handleSubnetDiscover processes subnet discovery results.
func (m Model) handleSubnetDiscover(msg subnetDiscoverMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = fmt.Sprintf("Subnet discovery failed: %v", msg.err)
		return m, nil
	}
	m.subnets = msg.subnets
	if len(m.subnets) == 0 {
		return m, nil
	}

	if m.subnet == "" {
		m.subnet = m.subnets[0].Subnet
		m.subnetIdx = 0
		m.status = fmt.Sprintf("Ready - Press 's' to scan (%d subnet(s) found, active: %s)", len(m.subnets), m.subnet)
		return m, nil
	}

	for i, subnet := range m.subnets {
		if subnet.Subnet == m.subnet {
			m.subnetIdx = i
			break
		}
	}
	return m, nil
}

// handleKeyMsg routes key messages to the appropriate view handler.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentView == ViewToolView {
		return m.updateToolView(msg)
	}
	if m.currentView == ViewScriptConsole {
		return m.updateScriptConsole(msg)
	}

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
		m.status = "Tool View - Select a tool (5-9)"
	case "4":
		m.currentView = ViewScriptConsole
		m.status = "Script Console - Enter command"
		m.scriptFocused = true
		m.scriptInput.Focus()
	case "s":
		return m.handleScanKey()
	case "n":
		return m.cycleSubnet()
	case "r":
		m.status = "Refreshing..."
	}

	if m.currentView == ViewDeviceList {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleScanKey handles the 's' key press to initiate a scan.
func (m Model) handleScanKey() (tea.Model, tea.Cmd) {
	if m.scanning {
		m.status = "Scan already in progress..."
		return m, nil
	}
	if m.orchestrator == nil {
		m.status = "Scanner not configured"
		return m, nil
	}
	if m.subnet == "" {
		m.status = "No subnet configured"
		return m, nil
	}
	m.scanning = true
	m.status = fmt.Sprintf("Scanning %s...", m.subnet)
	return m, m.startScan()
}

// cycleSubnet cycles through discovered subnets.
func (m Model) cycleSubnet() (tea.Model, tea.Cmd) {
	if m.scanning {
		m.status = "Cannot switch subnet during scan"
		return m, nil
	}
	if len(m.subnets) == 0 {
		m.status = "No subnets discovered"
		return m, nil
	}
	m.subnetIdx = (m.subnetIdx + 1) % len(m.subnets)
	m.subnet = m.subnets[m.subnetIdx].Subnet
	m.status = fmt.Sprintf("Active subnet: %s (%d/%d)", m.subnet, m.subnetIdx+1, len(m.subnets))
	return m, nil
}

// handleToolResult processes tool execution output.
func (m Model) handleToolResult(msg toolResultMsg) (tea.Model, tea.Cmd) {
	m.toolOutputText += msg.output
	m.toolOutput.SetContent(m.toolOutputText)
	m.toolOutput.GotoBottom()
	if msg.done {
		m.toolRunning = false
		m.status = "Tool execution completed"
	}
	return m, nil
}

// handleScriptResult processes script execution output.
func (m Model) handleScriptResult(msg scriptResultMsg) (tea.Model, tea.Cmd) {
	m.scriptRunning = false
	if msg.err != nil {
		m.scriptOutputText += fmt.Sprintf("Error: %v\n", msg.err)
	} else if msg.output != "" {
		m.scriptOutputText += msg.output
	} else {
		m.scriptOutputText += "Script completed successfully.\n"
	}
	m.scriptOutput.SetContent(m.scriptOutputText)
	m.scriptOutput.GotoBottom()
	m.status = "Script execution completed"
	return m, nil
}

// handleScanResult processes scan completion and updates state.
func (m Model) handleScanResult(msg scanResultMsg) (tea.Model, tea.Cmd) {
	m.scanning = false
	if msg.err != nil {
		m.status = fmt.Sprintf("Scan error: %v", msg.err)
		return m, nil
	}

	// Update registry with scan results to track state changes and generate alerts
	var newAlerts []tracker.Alert
	if m.registry != nil {
		_ = m.registry.Update(msg.result.Devices)
		newAlerts = m.registry.GetAlerts()
		m.alerts = append(m.alerts, newAlerts...)
		// Use registry's enriched devices (with status tracking)
		m.devices = m.registry.GetDevices()
	} else {
		m.devices = msg.result.Devices
	}
	m.scanResult = msg.result
	m.lastUpdate = time.Now()
	m.status = fmt.Sprintf("Scan complete: %d devices found in %v", len(m.devices), msg.result.ScanTime.Round(time.Millisecond))

	// Persist devices and new alerts to storage
	if m.storage != nil {
		if err := m.storage.SaveDevices(m.devices); err != nil {
			m.status += fmt.Sprintf(" (storage error: %v)", err)
		}
		for _, alert := range newAlerts {
			if err := m.storage.SaveAlert(alert); err != nil {
				m.status += fmt.Sprintf(" (alert save error: %v)", err)
				break
			}
		}
	}
	return m, nil
}

// handleWindowSize processes terminal resize events.
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.table.SetWidth(m.width - 4)
	m.table.SetHeight(m.height - 10)
	m.toolOutput.Width = m.width - 8
	m.toolOutput.Height = m.height - 18
	m.toolInput.Width = m.width - 20
	m.scriptOutput.Width = m.width - 8
	m.scriptOutput.Height = m.height - 18
	m.scriptInput.Width = m.width - 20
	m.ready = true
	return m, nil
}

// updateToolView handles input for the Tool View.
func (m Model) updateToolView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// If text input is focused, handle input
	if m.toolFocused {
		switch key {
		case "esc":
			m.toolFocused = false
			m.toolInput.Blur()
			m.status = "Tool View - Select a tool (1-5)"
			return m, nil
		case "enter":
			if !m.toolRunning && m.selectedTool >= 0 {
				return m.executeSelectedTool()
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.toolInput, cmd = m.toolInput.Update(msg)
		return m, cmd
	}

	// Handle tool selection and global keys
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1", "2", "3", "4":
		// View switching
		viewNum := int(key[0] - '1')
		m.currentView = ViewType(viewNum)
		m.status = []string{"Network Map View", "Device List View", "Tool View", "Script Console View"}[viewNum]
		return m, nil
	case "5", "6", "7", "8", "9":
		// Tool selection (when in tool view, 5-9 selects tools)
		toolNum := int(key[0] - '5')
		if toolNum < len(m.tools) {
			m.selectedTool = toolNum
			m.toolFocused = true
			m.toolInput.Focus()
			m.toolInput.SetValue("")
			m.status = fmt.Sprintf("Selected: %s - Enter arguments", m.tools[toolNum].Name())
		}
		return m, nil
	case "enter":
		// Enter focuses input if tool is selected
		if m.selectedTool >= 0 && !m.toolFocused {
			m.toolFocused = true
			m.toolInput.Focus()
			return m, nil
		}
	case "c":
		// Clear output
		m.toolOutputText = ""
		m.toolOutput.SetContent("Tool output will appear here.")
		return m, nil
	}

	return m, nil
}

// updateScriptConsole handles input for the Script Console view.
func (m Model) updateScriptConsole(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle global keys
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.scriptFocused = false
		m.scriptInput.Blur()
		m.status = "Script Console View"
		return m, nil
	}

	// If input is focused
	if m.scriptFocused {
		switch key {
		case "enter":
			return m.executeScriptCommand()
		}

		var cmd tea.Cmd
		m.scriptInput, cmd = m.scriptInput.Update(msg)
		return m, cmd
	}

	// Handle view switching if not focused
	switch key {
	case "q":
		return m, tea.Quit
	case "1", "2", "3", "4":
		viewNum := int(key[0] - '1')
		m.currentView = ViewType(viewNum)
		m.status = []string{"Network Map View", "Device List View", "Tool View", "Script Console View"}[viewNum]
		return m, nil
	case "enter":
		m.scriptFocused = true
		m.scriptInput.Focus()
		return m, nil
	case "c":
		m.scriptOutputText = ""
		m.scriptOutput.SetContent("Script output will appear here.")
		return m, nil
	}

	return m, nil
}

// executeScriptCommand parses and executes script console commands.
func (m Model) executeScriptCommand() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.scriptInput.Value())
	m.scriptInput.SetValue("")

	if input == "" {
		return m, nil
	}

	m.scriptOutputText += fmt.Sprintf("> %s\n", input)
	m.scriptOutput.SetContent(m.scriptOutputText)

	// Parse command
	if strings.HasPrefix(input, ":") {
		parts := strings.Fields(input)
		cmd := parts[0]

		switch cmd {
		case ":list":
			return m.listScripts()
		case ":load":
			if len(parts) < 2 {
				m.scriptOutputText += "Usage: :load <script.tengo>\n"
				m.scriptOutput.SetContent(m.scriptOutputText)
				return m, nil
			}
			return m.loadScript(parts[1])
		case ":stop":
			if m.scriptRunning {
				m.scriptEngine.Stop()
				m.scriptOutputText += "Script stopped.\n"
			} else {
				m.scriptOutputText += "No script is running.\n"
			}
			m.scriptOutput.SetContent(m.scriptOutputText)
			return m, nil
		case ":help":
			m.scriptOutputText += "Commands:\n"
			m.scriptOutputText += "  :list           - List available scripts\n"
			m.scriptOutputText += "  :load <file>    - Load and run a script\n"
			m.scriptOutputText += "  :stop           - Stop running script\n"
			m.scriptOutputText += "  :help           - Show this help\n"
			m.scriptOutput.SetContent(m.scriptOutputText)
			return m, nil
		default:
			m.scriptOutputText += fmt.Sprintf("Unknown command: %s\n", cmd)
			m.scriptOutput.SetContent(m.scriptOutputText)
			return m, nil
		}
	}

	// Run as inline script
	m.scriptRunning = true
	m.status = "Running script..."
	return m, m.runScript(input)
}

// listScripts lists available scripts in the scripts directory.
func (m Model) listScripts() (tea.Model, tea.Cmd) {
	entries, err := os.ReadDir(m.scriptsDir)
	if err != nil {
		m.scriptOutputText += fmt.Sprintf("Cannot read scripts directory: %v\n", err)
		m.scriptOutputText += fmt.Sprintf("Directory: %s\n", m.scriptsDir)
		m.scriptOutput.SetContent(m.scriptOutputText)
		return m, nil
	}

	if len(entries) == 0 {
		m.scriptOutputText += "No scripts found.\n"
		m.scriptOutputText += fmt.Sprintf("Directory: %s\n", m.scriptsDir)
		m.scriptOutput.SetContent(m.scriptOutputText)
		return m, nil
	}

	m.scriptOutputText += "Available scripts:\n"
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tengo") {
			m.scriptOutputText += fmt.Sprintf("  %s\n", entry.Name())
		}
	}
	m.scriptOutput.SetContent(m.scriptOutputText)
	return m, nil
}

// loadScript loads and runs a script file.
func (m Model) loadScript(filename string) (tea.Model, tea.Cmd) {
	scriptPath := filepath.Join(m.scriptsDir, filename)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		m.scriptOutputText += fmt.Sprintf("Script not found: %s\n", filename)
		m.scriptOutput.SetContent(m.scriptOutputText)
		return m, nil
	}

	m.scriptRunning = true
	m.status = fmt.Sprintf("Running script: %s", filename)
	return m, m.runScriptFile(scriptPath)
}

// runScript runs an inline script.
func (m Model) runScript(source string) tea.Cmd {
	engine := m.scriptEngine
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := engine.Run(ctx, source)
		return scriptResultMsg{output: "", err: err}
	}
}

// runScriptFile runs a script from a file.
func (m Model) runScriptFile(path string) tea.Cmd {
	engine := m.scriptEngine
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := engine.LoadFile(ctx, path)
		return scriptResultMsg{output: "", err: err}
	}
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
	var builder strings.Builder
	builder.WriteString("Available Network Tools:\n\n")

	toolNames := []string{"netcat", "telnet", "traceroute", "dig", "whois"}
	for i, name := range toolNames {
		prefix := "  "
		if i == m.selectedTool {
			prefix = "▶ "
			name = m.styles.Active.Render(fmt.Sprintf("[%d] %s", i+5, name))
		} else {
			name = fmt.Sprintf("[%d] %s", i+5, name)
		}
		builder.WriteString(prefix + name + "\n")
	}

	builder.WriteString("\n")

	// Show input field if tool is selected
	if m.selectedTool >= 0 {
		builder.WriteString(fmt.Sprintf("Tool: %s\n", m.tools[m.selectedTool].Name()))
		builder.WriteString("Args: ")
		builder.WriteString(m.toolInput.View())
		builder.WriteString("\n\n")
	}

	// Show output area
	renderOutputArea(&builder, m.toolOutputText, m.toolOutput.View(), "Tool output will appear here.")

	if m.toolRunning {
		builder.WriteString("\n[Running...] Press Esc to cancel")
	} else {
		builder.WriteString("\nPress 5-9 to select tool | Enter to run | c to clear | Esc to cancel")
	}

	return m.styles.Border.Width(m.width - 4).Render(builder.String())
}

// renderScriptConsole renders the script console view.
func (m Model) renderScriptConsole() string {
	var builder strings.Builder
	builder.WriteString("Script Console (Tengo):\n\n")
	builder.WriteString(fmt.Sprintf("Scripts directory: %s\n\n", m.scriptsDir))

	// Show input
	builder.WriteString("> ")
	builder.WriteString(m.scriptInput.View())
	builder.WriteString("\n\n")

	// Show output area
	renderOutputArea(&builder, m.scriptOutputText, m.scriptOutput.View(), "Script output will appear here.")

	if m.scriptRunning {
		builder.WriteString("\n[Running...] :stop to cancel")
	} else {
		builder.WriteString("\n:list | :load <file> | :stop | :help | c to clear | Esc to unfocus")
	}

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
		"q: quit | s: scan | n: next subnet | r: refresh | 1-4: switch views | ↑↓: navigate",
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

// executeSelectedTool runs the selected tool with provided arguments.
func (m Model) executeSelectedTool() (tea.Model, tea.Cmd) {
	if m.selectedTool < 0 || m.selectedTool >= len(m.tools) {
		m.status = "No tool selected"
		return m, nil
	}

	args := strings.Fields(m.toolInput.Value())
	tool := m.tools[m.selectedTool]

	// Validate args
	if err := tool.Validate(args); err != nil {
		m.toolOutputText += fmt.Sprintf("Error: %v\n", err)
		m.toolOutput.SetContent(m.toolOutputText)
		return m, nil
	}

	m.toolRunning = true
	m.toolOutputText += fmt.Sprintf(">>> %s %s\n", tool.Name(), strings.Join(args, " "))
	m.toolOutput.SetContent(m.toolOutputText)
	m.status = fmt.Sprintf("Running %s...", tool.Name())
	m.toolFocused = false
	m.toolInput.Blur()

	return m, m.runTool(tool, args)
}

// runTool returns a command that executes the tool asynchronously.
func (m Model) runTool(tool tools.NetworkTool, args []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		outputChan, err := tool.Execute(ctx, args)
		if err != nil {
			return toolResultMsg{output: fmt.Sprintf("Error: %v\n", err), done: true}
		}

		var result strings.Builder
		for line := range outputChan {
			result.WriteString(line)
		}

		return toolResultMsg{output: result.String(), done: true}
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

// RunWithOrchestratorAndStorage starts the TUI with a scanner orchestrator and storage.
func RunWithOrchestratorAndStorage(orch *scanner.Orchestrator, subnet string, storage *tracker.Storage) error {
	p := tea.NewProgram(NewModelWithOrchestratorAndStorage(orch, subnet, storage), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// devicesToMaps converts a slice of scanner.Device to a slice of maps for script API use.
func devicesToMaps(devices []scanner.Device) []map[string]interface{} {
	result := make([]map[string]interface{}, len(devices))
	for i, d := range devices {
		m := map[string]interface{}{
			"ip":       d.IP.String(),
			"hostname": d.Hostname,
			"vendor":   d.Vendor,
			"status":   string(d.Status),
		}
		if d.MAC != nil {
			m["mac"] = d.MAC.String()
		}
		if len(d.Ports) > 0 {
			ports := make([]interface{}, len(d.Ports))
			for j, p := range d.Ports {
				ports[j] = p
			}
			m["ports"] = ports
		}
		result[i] = m
	}
	return result
}

// newInputAndViewport creates a text input and viewport pair with the given settings.
func newInputAndViewport(placeholder, defaultContent string) (textinput.Model, viewport.Model) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.Width = 50

	vp := viewport.New(60, 10)
	vp.SetContent(defaultContent)

	return ti, vp
}

// renderOutputArea writes a bordered output section to the builder.
func renderOutputArea(builder *strings.Builder, text, viewContent, placeholder string) {
	builder.WriteString("Output:\n")
	builder.WriteString("─────────────────────────────────────────────\n")
	if text != "" {
		builder.WriteString(viewContent)
	} else {
		builder.WriteString(placeholder + "\n")
	}
	builder.WriteString("\n─────────────────────────────────────────────\n")
}
