package tui

import (
	"net"
	"strings"
	"testing"

	"github.com/opd-ai/tuimap/internal/scanner"
)

func TestDeviceRoleString(t *testing.T) {
	tests := []struct {
		role DeviceRole
		want string
	}{
		{RoleGateway, "Gateway"},
		{RoleRouter, "Router"},
		{RoleClient, "Client"},
		{DeviceRole(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("DeviceRole(%d).String() = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

func TestBuildLabel(t *testing.T) {
	tests := []struct {
		name   string
		device scanner.Device
		want   string
	}{
		{
			name:   "uses hostname when available",
			device: scanner.Device{IP: net.ParseIP("192.168.1.1"), Hostname: "my-router"},
			want:   "my-router",
		},
		{
			name:   "falls back to IP when no hostname",
			device: scanner.Device{IP: net.ParseIP("192.168.1.50")},
			want:   "192.168.1.50",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildLabel(tt.device); got != tt.want {
				t.Errorf("buildLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyDevice(t *testing.T) {
	gatewayIP := net.ParseIP("192.168.1.1")

	tests := []struct {
		name   string
		device scanner.Device
		want   DeviceRole
	}{
		{
			name:   "matches gateway IP",
			device: scanner.Device{IP: net.ParseIP("192.168.1.1")},
			want:   RoleGateway,
		},
		{
			name:   "router by hostname",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Hostname: "office-router"},
			want:   RoleRouter,
		},
		{
			name:   "router by vendor",
			device: scanner.Device{IP: net.ParseIP("192.168.1.3"), Vendor: "Cisco Systems"},
			want:   RoleRouter,
		},
		{
			name:   "router by ports",
			device: scanner.Device{IP: net.ParseIP("192.168.1.4"), Ports: []int{53, 67, 80}},
			want:   RoleRouter,
		},
		{
			name:   "router by .1 IP suffix",
			device: scanner.Device{IP: net.ParseIP("10.0.0.1")},
			want:   RoleRouter,
		},
		{
			name:   "client device",
			device: scanner.Device{IP: net.ParseIP("192.168.1.100"), Hostname: "laptop"},
			want:   RoleClient,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDevice(tt.device, gatewayIP); got != tt.want {
				t.Errorf("classifyDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyDeviceNilGateway(t *testing.T) {
	device := scanner.Device{IP: net.ParseIP("192.168.1.100"), Hostname: "laptop"}
	got := classifyDevice(device, nil)
	if got != RoleClient {
		t.Errorf("classifyDevice with nil gateway = %v, want RoleClient", got)
	}
}

func TestIsLikelyRouter(t *testing.T) {
	tests := []struct {
		name   string
		device scanner.Device
		want   bool
	}{
		{
			name:   "hostname contains router",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Hostname: "my-Router"},
			want:   true,
		},
		{
			name:   "hostname contains gateway",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Hostname: "gw.local"},
			want:   true,
		},
		{
			name:   "hostname contains firewall",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Hostname: "firewall-01"},
			want:   true,
		},
		{
			name:   "vendor is ubiquiti",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Vendor: "Ubiquiti Networks"},
			want:   true,
		},
		{
			name:   "vendor is mikrotik",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Vendor: "MikroTik"},
			want:   true,
		},
		{
			name:   "has DNS and DHCP ports",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Ports: []int{53, 67}},
			want:   true,
		},
		{
			name:   "only one router port is not enough",
			device: scanner.Device{IP: net.ParseIP("192.168.1.2"), Ports: []int{53, 80}},
			want:   false,
		},
		{
			name:   "IP ends in .1",
			device: scanner.Device{IP: net.ParseIP("172.16.0.1")},
			want:   true,
		},
		{
			name:   "normal client",
			device: scanner.Device{IP: net.ParseIP("192.168.1.50"), Hostname: "laptop"},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLikelyRouter(tt.device); got != tt.want {
				t.Errorf("isLikelyRouter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyDevices(t *testing.T) {
	gatewayIP := net.ParseIP("192.168.1.1")
	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.100"), Hostname: "laptop", Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.1"), Hostname: "router", Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.2"), Hostname: "switch", Vendor: "Cisco Systems", Status: scanner.StatusOnline},
	}

	nodes := classifyDevices(devices, gatewayIP)

	if len(nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(nodes))
	}

	// Should be sorted: gateway first, then router, then client
	if nodes[0].Role != RoleGateway {
		t.Errorf("First node should be gateway, got %v", nodes[0].Role)
	}
	if nodes[1].Role != RoleRouter {
		t.Errorf("Second node should be router, got %v", nodes[1].Role)
	}
	if nodes[2].Role != RoleClient {
		t.Errorf("Third node should be client, got %v", nodes[2].Role)
	}
}

func TestPartitionNodes(t *testing.T) {
	nodes := []NetworkNode{
		{Role: RoleGateway, Label: "gw"},
		{Role: RoleRouter, Label: "r1"},
		{Role: RoleRouter, Label: "r2"},
		{Role: RoleClient, Label: "c1"},
		{Role: RoleClient, Label: "c2"},
		{Role: RoleClient, Label: "c3"},
	}

	gw, r, c := partitionNodes(nodes)
	if len(gw) != 1 {
		t.Errorf("Expected 1 gateway, got %d", len(gw))
	}
	if len(r) != 2 {
		t.Errorf("Expected 2 routers, got %d", len(r))
	}
	if len(c) != 3 {
		t.Errorf("Expected 3 clients, got %d", len(c))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"ab", 1, "…"},
		{"", 5, ""},
		{"日本語テスト", 4, "日本語…"},
		{"日本語テスト", 3, "日本…"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncate(tt.input, tt.maxLen); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name    string
		ports   []int
		maxShow int
		want    string
	}{
		{"empty", nil, 3, ""},
		{"one port", []int{80}, 3, "80"},
		{"within limit", []int{443, 80}, 3, "80,443"},
		{"exceeds limit sorted", []int{8443, 80, 443, 8080}, 3, "80,443,8080+1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatPorts(tt.ports, tt.maxShow); got != tt.want {
				t.Errorf("formatPorts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPortsDoesNotMutateInput(t *testing.T) {
	ports := []int{8080, 22, 443}
	_ = formatPorts(ports, 2)
	if ports[0] != 8080 || ports[1] != 22 || ports[2] != 443 {
		t.Errorf("formatPorts mutated input slice: %v", ports)
	}
}

func TestStatusIndicator(t *testing.T) {
	ds := newDiagramStyles()

	tests := []struct {
		status scanner.DeviceStatus
		expect string
	}{
		{scanner.StatusOnline, "●"},
		{scanner.StatusOffline, "○"},
		{scanner.StatusNew, "★"},
		{scanner.StatusChanged, "△"},
		{scanner.DeviceStatus("unknown"), "?"},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := statusIndicator(tt.status, ds)
			if !strings.Contains(got, tt.expect) {
				t.Errorf("statusIndicator(%q) = %q, expected to contain %q", tt.status, got, tt.expect)
			}
		})
	}
}

func TestRenderDiagramEmpty(t *testing.T) {
	result := renderDiagram(nil, nil, 80)
	if !strings.Contains(result, "No devices discovered") {
		t.Error("Expected empty state message")
	}
}

func TestRenderDiagramWithDevices(t *testing.T) {
	gatewayIP := net.ParseIP("192.168.1.1")
	scanResult := &scanner.ScanResult{
		NetworkInfo: scanner.NetworkMetadata{
			Gateway: gatewayIP,
			Subnet:  "192.168.1.0/24",
		},
	}

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Hostname: "router", Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.100"), Hostname: "laptop", Status: scanner.StatusOnline, Ports: []int{22, 80}},
		{IP: net.ParseIP("192.168.1.101"), Status: scanner.StatusOffline},
		{IP: net.ParseIP("192.168.1.200"), Status: scanner.StatusNew},
	}

	result := renderDiagram(devices, scanResult, 80)

	// Check structure
	if !strings.Contains(result, "Network Topology") {
		t.Error("Expected header 'Network Topology'")
	}
	if !strings.Contains(result, "Internet") {
		t.Error("Expected internet cloud")
	}
	if !strings.Contains(result, "Gateway") {
		t.Error("Expected gateway section")
	}
	if !strings.Contains(result, "Clients") {
		t.Error("Expected clients section")
	}
	if !strings.Contains(result, "Legend") {
		t.Error("Expected legend")
	}
	if !strings.Contains(result, "Total:") {
		t.Error("Expected total summary")
	}
}

func TestRenderDiagramNoGateway(t *testing.T) {
	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.100"), Status: scanner.StatusOnline},
	}
	result := renderDiagram(devices, nil, 80)
	if !strings.Contains(result, "No gateway detected") {
		t.Error("Expected 'No gateway detected' message when no scan result")
	}
}

func TestRenderDiagramWithRouters(t *testing.T) {
	gatewayIP := net.ParseIP("192.168.1.1")
	scanResult := &scanner.ScanResult{
		NetworkInfo: scanner.NetworkMetadata{Gateway: gatewayIP},
	}

	devices := []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Hostname: "gw", Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.2"), Hostname: "switch", Vendor: "Cisco Systems", Status: scanner.StatusOnline},
		{IP: net.ParseIP("192.168.1.100"), Hostname: "laptop", Status: scanner.StatusOnline},
	}

	result := renderDiagram(devices, scanResult, 80)
	if !strings.Contains(result, "Routers") {
		t.Error("Expected router section when router-like devices are present")
	}
}

func TestRenderNetworkMapIntegration(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.width = 80
	m.height = 24

	// Empty case
	result := m.renderNetworkMap()
	if result == "" {
		t.Error("Expected non-empty network map for empty devices")
	}

	// With devices
	m.devices = []scanner.Device{
		{IP: net.ParseIP("192.168.1.1"), Status: scanner.StatusOnline, Hostname: "gw"},
		{IP: net.ParseIP("192.168.1.100"), Status: scanner.StatusNew},
	}
	m.scanResult = &scanner.ScanResult{
		NetworkInfo: scanner.NetworkMetadata{
			Gateway: net.ParseIP("192.168.1.1"),
		},
	}

	result = m.renderNetworkMap()
	if result == "" {
		t.Error("Expected non-empty network map with devices")
	}
	if !strings.Contains(result, "Internet") {
		t.Error("Expected internet cloud in network map")
	}
}

func TestNewDiagramStyles(t *testing.T) {
	ds := newDiagramStyles()

	// Verify all styles are usable without panic
	_ = ds.gateway.Render("test")
	_ = ds.router.Render("test")
	_ = ds.client.Render("test")
	_ = ds.online.Render("test")
	_ = ds.offline.Render("test")
	_ = ds.newDev.Render("test")
	_ = ds.changed.Render("test")
	_ = ds.line.Render("test")
	_ = ds.dimmed.Render("test")
	_ = ds.header.Render("test")
	_ = ds.roleTag.Render("test")
	_ = ds.internet.Render("test")
}

func TestRenderDiagramAllStatuses(t *testing.T) {
	devices := []scanner.Device{
		{IP: net.ParseIP("10.0.0.2"), Status: scanner.StatusOnline},
		{IP: net.ParseIP("10.0.0.3"), Status: scanner.StatusOffline},
		{IP: net.ParseIP("10.0.0.4"), Status: scanner.StatusNew},
		{IP: net.ParseIP("10.0.0.5"), Status: scanner.StatusChanged},
	}
	result := renderDiagram(devices, nil, 80)
	if !strings.Contains(result, "●") {
		t.Error("Expected online indicator")
	}
	if !strings.Contains(result, "○") {
		t.Error("Expected offline indicator")
	}
	if !strings.Contains(result, "★") {
		t.Error("Expected new indicator")
	}
	if !strings.Contains(result, "△") {
		t.Error("Expected changed indicator")
	}
	// Legend should include changed status
	if !strings.Contains(result, "changed") {
		t.Error("Expected 'changed' in legend")
	}
}

func TestRenderDiagramPortDisplay(t *testing.T) {
	devices := []scanner.Device{
		{IP: net.ParseIP("10.0.0.100"), Status: scanner.StatusOnline, Ports: []int{22, 80, 443, 8080, 8443}},
	}
	result := renderDiagram(devices, nil, 80)
	if !strings.Contains(result, "22") {
		t.Error("Expected port 22 in output")
	}
}
