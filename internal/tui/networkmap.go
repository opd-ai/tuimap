// Package tui provides the Bubble Tea terminal user interface.
package tui

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/opd-ai/tuimap/internal/scanner"
)

// DeviceRole represents the role of a device in the network topology.
type DeviceRole int

const (
	RoleGateway DeviceRole = iota
	RoleRouter
	RoleClient
)

// String returns a human-readable label for the device role.
func (r DeviceRole) String() string {
	switch r {
	case RoleGateway:
		return "Gateway"
	case RoleRouter:
		return "Router"
	case RoleClient:
		return "Client"
	default:
		return "Unknown"
	}
}

// NetworkNode represents a device in the network diagram with its role.
type NetworkNode struct {
	Device scanner.Device
	Role   DeviceRole
	Label  string // Display label (hostname or IP)
}

// classifyDevices assigns roles to devices based on network metadata and heuristics.
func classifyDevices(devices []scanner.Device, gatewayIP net.IP) []NetworkNode {
	nodes := make([]NetworkNode, 0, len(devices))

	for _, device := range devices {
		node := NetworkNode{
			Device: device,
			Label:  buildLabel(device),
		}
		node.Role = classifyDevice(device, gatewayIP)
		nodes = append(nodes, node)
	}

	// Sort: gateway first, then routers, then clients
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Role != nodes[j].Role {
			return nodes[i].Role < nodes[j].Role
		}
		return nodes[i].Label < nodes[j].Label
	})

	return nodes
}

// classifyDevice determines the role of a single device.
func classifyDevice(device scanner.Device, gatewayIP net.IP) DeviceRole {
	// Check if this is the gateway
	if gatewayIP != nil && device.IP.Equal(gatewayIP) {
		return RoleGateway
	}

	// Heuristic: common router indicators
	if isLikelyRouter(device) {
		return RoleRouter
	}

	return RoleClient
}

// isLikelyRouter uses heuristics to detect router-like devices.
func isLikelyRouter(device scanner.Device) bool {
	// Check hostname for router-like patterns using word boundary matching
	// to avoid false positives (e.g. "laptop" matching "ap").
	hostname := strings.ToLower(device.Hostname)
	routerKeywords := []string{"router", "gateway", "gw", "ap", "switch", "fw", "firewall"}
	for _, keyword := range routerKeywords {
		if matchWordBoundary(hostname, keyword) {
			return true
		}
	}

	// Check vendor for known router manufacturers
	vendor := strings.ToLower(device.Vendor)
	routerVendors := []string{
		"cisco", "juniper", "mikrotik", "ubiquiti", "netgear", "tp-link",
		"tplink", "linksys", "asus", "d-link", "dlink", "aruba", "fortinet",
		"pfsense", "openwrt", "meraki",
	}
	for _, rv := range routerVendors {
		if strings.Contains(vendor, rv) {
			return true
		}
	}

	// Check for router/network-infrastructure ports (DNS, DHCP, SNMP, BGP)
	routerPorts := map[int]bool{53: true, 67: true, 68: true, 161: true, 179: true}
	routerPortCount := 0
	for _, port := range device.Ports {
		if routerPorts[port] {
			routerPortCount++
		}
	}
	// If device has 2+ router-specific ports, likely a router
	if routerPortCount >= 2 {
		return true
	}

	// Check if IP ends in .1 (common for gateways/routers in subnets)
	ip4 := device.IP.To4()
	if ip4 != nil && ip4[3] == 1 {
		return true
	}

	return false
}

// buildLabel creates a display label for a device.
func buildLabel(device scanner.Device) string {
	if device.Hostname != "" {
		return device.Hostname
	}
	return device.IP.String()
}

// diagramStyles holds styles used for network diagram rendering.
type diagramStyles struct {
	gateway  lipgloss.Style
	router   lipgloss.Style
	client   lipgloss.Style
	online   lipgloss.Style
	offline  lipgloss.Style
	newDev   lipgloss.Style
	changed  lipgloss.Style
	line     lipgloss.Style
	dimmed   lipgloss.Style
	header   lipgloss.Style
	roleTag  lipgloss.Style
	internet lipgloss.Style
}

func newDiagramStyles() diagramStyles {
	return diagramStyles{
		gateway: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		router: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81")),
		client: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		online: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")),
		offline: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		newDev: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")),
		changed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
		line: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		dimmed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		roleTag: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		internet: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
	}
}

// renderDiagram renders a full network topology diagram given devices and
// optional scan result metadata.
func renderDiagram(devices []scanner.Device, scanResult *scanner.ScanResult, width int) string {
	if len(devices) == 0 {
		return "  No devices discovered yet.\n  Press 's' to scan the network."
	}

	ds := newDiagramStyles()

	var gatewayIP net.IP
	if scanResult != nil {
		gatewayIP = scanResult.NetworkInfo.Gateway
	}

	nodes := classifyDevices(devices, gatewayIP)

	var b strings.Builder

	// Header
	b.WriteString(ds.header.Render("Network Topology"))
	b.WriteString("\n\n")

	// Internet cloud
	b.WriteString(renderInternet(ds))
	b.WriteString("\n")

	// Vertical connector
	b.WriteString(ds.line.Render("          ║"))
	b.WriteString("\n")

	// Render gateway(s)
	gateways, routers, clients := partitionNodes(nodes)

	if len(gateways) > 0 {
		for _, gw := range gateways {
			b.WriteString(renderGatewayNode(gw, ds))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(ds.dimmed.Render("    [No gateway detected]"))
		b.WriteString("\n")
	}

	// Trunk line from gateway
	b.WriteString(ds.line.Render("          ║"))
	b.WriteString("\n")

	// Render routers if any
	if len(routers) > 0 {
		b.WriteString(renderRouterSection(routers, ds))
		b.WriteString("\n")
		b.WriteString(ds.line.Render("          ║"))
		b.WriteString("\n")
	}

	// Render clients
	if len(clients) > 0 {
		b.WriteString(renderClientSection(clients, ds, width))
	} else {
		b.WriteString(ds.dimmed.Render("    [No client devices]"))
		b.WriteString("\n")
	}

	// Legend
	b.WriteString("\n")
	b.WriteString(renderLegend(ds))

	// Summary line
	b.WriteString("\n")
	b.WriteString(ds.dimmed.Render(fmt.Sprintf(
		"  Total: %d devices (%d gateways, %d routers, %d clients)",
		len(nodes), len(gateways), len(routers), len(clients),
	)))
	b.WriteString("\n")

	return b.String()
}

// renderInternet draws a small internet cloud.
func renderInternet(ds diagramStyles) string {
	return ds.internet.Render("    ☁  Internet")
}

// gatewayBoxMinWidth is the minimum character width inside the gateway box.
// The actual box width is computed from the rendered content width so the
// border always matches the visible text, while labels are still truncated to
// gatewayLabelMaxLen to avoid unnecessary growth.
const gatewayBoxMinWidth = 22
const gatewayLabelMaxLen = 14

// padRightWidth pads s with spaces on the right so its visible width equals width.
func padRightWidth(s string, width int) string {
	padding := width - lipgloss.Width(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

// renderGatewayNode draws a single gateway node with box.
func renderGatewayNode(node NetworkNode, ds diagramStyles) string {
	statusIcon := statusIndicator(node.Device.Status, ds)
	ip := node.Device.IP.String()
	label := ds.gateway.Render(truncate(node.Label, gatewayLabelMaxLen))
	role := ds.roleTag.Render("[Gateway]")

	content := fmt.Sprintf("%s %s %s", statusIcon, label, role)
	boxInnerWidth := lipgloss.Width(content) + 2
	if boxInnerWidth < gatewayBoxMinWidth {
		boxInnerWidth = gatewayBoxMinWidth
	}

	border := strings.Repeat("═", boxInnerWidth)
	paddedContent := padRightWidth(content, boxInnerWidth-2)

	return fmt.Sprintf("    %s ╔%s╗\n      ║ %s ║\n      ╚%s╝",
		ds.line.Render("  "),
		border,
		paddedContent,
		border,
	) + "\n" + ds.dimmed.Render(fmt.Sprintf("          %s", ip))
}

// renderRouterSection draws the router tier.
func renderRouterSection(routers []NetworkNode, ds diagramStyles) string {
	var b strings.Builder
	b.WriteString(ds.header.Render("    ─── Routers/APs ───"))
	b.WriteString("\n")

	for i, r := range routers {
		statusIcon := statusIndicator(r.Device.Status, ds)
		ip := r.Device.IP.String()
		prefix := ds.line.Render("    ├── ")
		if i == len(routers)-1 {
			prefix = ds.line.Render("    └── ")
		}
		fmt.Fprintf(&b, "%s%s %s %s\n",
			prefix,
			statusIcon,
			ds.router.Render(truncate(r.Label, 18)),
			ds.dimmed.Render(ip),
		)
	}

	return b.String()
}

// renderClientSection draws the client tier in a compact layout.
func renderClientSection(clients []NetworkNode, ds diagramStyles, width int) string {
	var b strings.Builder
	b.WriteString(ds.header.Render("    ─── Clients ───"))
	b.WriteString("\n")

	for i, c := range clients {
		statusIcon := statusIndicator(c.Device.Status, ds)
		ip := c.Device.IP.String()

		prefix := ds.line.Render("    ├── ")
		if i == len(clients)-1 {
			prefix = ds.line.Render("    └── ")
		}

		info := ds.client.Render(truncate(c.Label, 18))
		fmt.Fprintf(&b, "%s%s %s %s",
			prefix,
			statusIcon,
			info,
			ds.dimmed.Render(ip),
		)

		// Add port info if available
		if len(c.Device.Ports) > 0 {
			portStr := formatPorts(c.Device.Ports, 3)
			b.WriteString(" " + ds.dimmed.Render("["+portStr+"]"))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// renderLegend draws the symbol legend.
func renderLegend(ds diagramStyles) string {
	return ds.dimmed.Render(
		"  Legend: " +
			ds.online.Render("●") + " online  " +
			ds.offline.Render("○") + " offline  " +
			ds.newDev.Render("★") + " new  " +
			ds.changed.Render("△") + " changed",
	)
}

// statusIndicator returns the status icon for a device.
func statusIndicator(status scanner.DeviceStatus, ds diagramStyles) string {
	switch status {
	case scanner.StatusOnline:
		return ds.online.Render("●")
	case scanner.StatusOffline:
		return ds.offline.Render("○")
	case scanner.StatusNew:
		return ds.newDev.Render("★")
	case scanner.StatusChanged:
		return ds.changed.Render("△")
	default:
		return ds.dimmed.Render("?")
	}
}

// partitionNodes splits nodes by role.
func partitionNodes(nodes []NetworkNode) (gateways, routers, clients []NetworkNode) {
	for _, n := range nodes {
		switch n.Role {
		case RoleGateway:
			gateways = append(gateways, n)
		case RoleRouter:
			routers = append(routers, n)
		case RoleClient:
			clients = append(clients, n)
		}
	}
	return
}

// truncate limits a string to maxLen runes, appending "…" if truncated.
// It operates on rune count rather than byte length to avoid splitting
// multi-byte UTF-8 characters.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// matchWordBoundary checks if keyword appears in s as a whole word,
// delimited by start/end of string, hyphens, dots, or underscores.
func matchWordBoundary(s, keyword string) bool {
	idx := 0
	for {
		pos := strings.Index(s[idx:], keyword)
		if pos < 0 {
			return false
		}
		start := idx + pos
		end := start + len(keyword)

		startOK := start == 0 || isBoundary(s[start-1])
		endOK := end == len(s) || isBoundary(s[end])

		if startOK && endOK {
			return true
		}
		idx = start + 1
		if idx >= len(s) {
			return false
		}
	}
}

// isBoundary returns true if the byte is a word boundary character.
func isBoundary(b byte) bool {
	return b == '-' || b == '.' || b == '_' || b == ' '
}

// formatPorts returns a compact string of the first n ports, sorted
// numerically for stable display regardless of scan order.
func formatPorts(ports []int, maxShow int) string {
	if len(ports) == 0 {
		return ""
	}
	sorted := make([]int, len(ports))
	copy(sorted, ports)
	sort.Ints(sorted)

	show := sorted
	suffix := ""
	if len(sorted) > maxShow {
		show = sorted[:maxShow]
		suffix = fmt.Sprintf("+%d", len(sorted)-maxShow)
	}
	strs := make([]string, len(show))
	for i, p := range show {
		strs[i] = fmt.Sprintf("%d", p)
	}
	result := strings.Join(strs, ",")
	if suffix != "" {
		result += suffix
	}
	return result
}
