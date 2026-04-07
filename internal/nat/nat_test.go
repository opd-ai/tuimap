package nat

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNATTypeConstants(t *testing.T) {
	tests := []struct {
		natType  NATType
		expected string
	}{
		{NATTypeNone, "none"},
		{NATTypeFull, "full_cone"},
		{NATTypeRestricted, "restricted_cone"},
		{NATTypePortRestricted, "port_restricted"},
		{NATTypeSymmetric, "symmetric"},
		{NATTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.natType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.natType)
		}
	}
}

func TestProtocolConstants(t *testing.T) {
	if string(ProtocolTCP) != "tcp" {
		t.Errorf("Expected tcp, got %s", ProtocolTCP)
	}
	if string(ProtocolUDP) != "udp" {
		t.Errorf("Expected udp, got %s", ProtocolUDP)
	}
}

func TestNewClient(t *testing.T) {
	// Test with default STUN servers
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if len(client.stunServers) == 0 {
		t.Error("Expected default STUN servers")
	}
	if client.mappings == nil {
		t.Error("Mappings map not initialized")
	}

	// Test with custom STUN servers
	customServers := []string{"stun.example.com:3478"}
	client2 := NewClient(customServers...)
	if len(client2.stunServers) != 1 || client2.stunServers[0] != "stun.example.com:3478" {
		t.Error("Custom STUN servers not set correctly")
	}
}

func TestNATInfoStruct(t *testing.T) {
	info := NATInfo{
		ExternalIP:   net.ParseIP("203.0.113.1"),
		InternalIP:   net.ParseIP("192.168.1.100"),
		GatewayIP:    net.ParseIP("192.168.1.1"),
		Type:         NATTypeRestricted,
		UPnPEnabled:  true,
		NATPMPEnable: false,
		Latency:      50 * time.Millisecond,
	}

	if info.ExternalIP.String() != "203.0.113.1" {
		t.Errorf("Expected external IP 203.0.113.1, got %s", info.ExternalIP)
	}
	if info.Type != NATTypeRestricted {
		t.Errorf("Expected NAT type restricted_cone, got %s", info.Type)
	}
	if !info.UPnPEnabled {
		t.Error("Expected UPnP to be enabled")
	}
}

func TestPortMappingStruct(t *testing.T) {
	mapping := PortMapping{
		InternalPort: 8080,
		ExternalPort: 80,
		Protocol:     ProtocolTCP,
		Description:  "Web server",
		Lifetime:     1 * time.Hour,
		CreatedAt:    time.Now(),
	}

	if mapping.InternalPort != 8080 {
		t.Errorf("Expected internal port 8080, got %d", mapping.InternalPort)
	}
	if mapping.Protocol != ProtocolTCP {
		t.Errorf("Expected protocol tcp, got %s", mapping.Protocol)
	}
}

func TestListMappings(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	mappings, err := client.ListMappings(ctx)
	if err != nil {
		t.Fatalf("ListMappings failed: %v", err)
	}

	if len(mappings) != 0 {
		t.Errorf("Expected 0 mappings initially, got %d", len(mappings))
	}
}

func TestAddPortMappingInvalidProtocol(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	_, err := client.AddPortMapping(ctx, 8080, 80, "invalid", "test", time.Hour)
	if err != ErrInvalidProtocol {
		t.Errorf("Expected ErrInvalidProtocol, got %v", err)
	}
}

func TestRemovePortMapping(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// RemovePortMapping should not error even if mapping doesn't exist
	err := client.RemovePortMapping(ctx, 80, ProtocolTCP)
	if err != nil {
		t.Errorf("RemovePortMapping should not error: %v", err)
	}
}

func TestGetLocalIP(t *testing.T) {
	ip, err := getLocalIP()
	if err != nil {
		t.Skipf("Could not get local IP (may be expected in isolated environment): %v", err)
	}

	if ip == nil {
		t.Error("getLocalIP returned nil")
	}

	// Verify it's a valid IP
	if ip.To4() == nil && ip.To16() == nil {
		t.Error("getLocalIP returned invalid IP")
	}
}

func TestGetDefaultGateway(t *testing.T) {
	gateway, err := getDefaultGateway()
	if err != nil {
		t.Skipf("Could not get default gateway (may be expected in isolated environment): %v", err)
	}

	if gateway == nil {
		t.Error("getDefaultGateway returned nil")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"hello", "hello", true},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
		{"HTTP/1.1 200 OK", "200 OK", true},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestParseSTUNResponseTooShort(t *testing.T) {
	_, err := parseSTUNResponse([]byte{0x01, 0x01})
	if err == nil {
		t.Error("Expected error for short response")
	}
}

func TestParseSTUNResponseNotBindingResponse(t *testing.T) {
	data := make([]byte, 20)
	data[0] = 0x00 // Not a binding response
	data[1] = 0x01

	_, err := parseSTUNResponse(data)
	if err == nil {
		t.Error("Expected error for non-binding response")
	}
}

func TestParseSTUNResponseValid(t *testing.T) {
	// Construct a valid STUN binding response with XOR-MAPPED-ADDRESS
	data := make([]byte, 32)
	data[0] = 0x01 // Binding Response
	data[1] = 0x01
	data[2] = 0x00 // Length high byte
	data[3] = 0x0c // Length: 12 bytes (one XOR-MAPPED-ADDRESS attribute)

	// Magic Cookie
	data[4] = 0x21
	data[5] = 0x12
	data[6] = 0xa4
	data[7] = 0x42

	// Transaction ID (8-19)
	for i := 8; i < 20; i++ {
		data[i] = byte(i)
	}

	// XOR-MAPPED-ADDRESS attribute
	data[20] = 0x00 // Type high
	data[21] = 0x20 // Type low (0x0020)
	data[22] = 0x00 // Length high
	data[23] = 0x08 // Length low (8 bytes)
	data[24] = 0x00 // Reserved
	data[25] = 0x01 // Family: IPv4
	// XOR'd port (skip)
	data[26] = 0x00
	data[27] = 0x00
	// XOR'd IP: 192.168.1.100 XOR 0x2112a442
	// 192.168.1.100 = 0xC0A80164
	// XOR with 0x2112A442 = 0xE1BAA526
	data[28] = 0xe1
	data[29] = 0xba
	data[30] = 0xa5
	data[31] = 0x26

	ip, err := parseSTUNResponse(data)
	if err != nil {
		t.Fatalf("parseSTUNResponse failed: %v", err)
	}

	expected := net.IPv4(192, 168, 1, 100)
	if !ip.Equal(expected) {
		t.Errorf("Expected IP %s, got %s", expected, ip)
	}
}

func TestDiscoverWithTimeout(t *testing.T) {
	client := NewClient("10.255.255.1:3478") // Non-routable address
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Discover should complete even with unreachable STUN server
	info, err := client.Discover(ctx)
	if err != nil {
		// Error is acceptable if network is unavailable
		t.Skipf("Discover returned error (may be expected): %v", err)
	}

	if info == nil {
		t.Error("Expected NATInfo, got nil")
	}
}

func TestGetExternalIPCached(t *testing.T) {
	client := NewClient()
	// Pre-set external IP
	client.externalIP = net.ParseIP("203.0.113.1")

	ctx := context.Background()
	ip, err := client.GetExternalIP(ctx)
	if err != nil {
		t.Fatalf("GetExternalIP failed: %v", err)
	}

	if !ip.Equal(net.ParseIP("203.0.113.1")) {
		t.Errorf("Expected cached IP, got %s", ip)
	}
}

func TestErrors(t *testing.T) {
	// Verify error variables are defined
	errors := []error{
		ErrNoNATDevice,
		ErrNATUnsupported,
		ErrPortMapFailed,
		ErrSTUNFailed,
		ErrTimeout,
		ErrInvalidProtocol,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error variable is nil")
		}
		if err.Error() == "" {
			t.Error("Error message is empty")
		}
	}
}
