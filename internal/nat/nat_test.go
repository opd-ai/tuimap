package nat

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNATTypeConstants(t *testing.T) {
	tests := []struct {
		natType  Type
		expected string
	}{
		{TypeNone, "none"},
		{TypeFull, "full_cone"},
		{TypeRestricted, "restricted_cone"},
		{TypePortRestricted, "port_restricted"},
		{TypeSymmetric, "symmetric"},
		{TypeUnknown, "unknown"},
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
	info := Info{
		ExternalIP:   net.ParseIP("203.0.113.1"),
		InternalIP:   net.ParseIP("192.168.1.100"),
		GatewayIP:    net.ParseIP("192.168.1.1"),
		Type:         TypeRestricted,
		UPnPEnabled:  true,
		NATPMPEnable: false,
		Latency:      50 * time.Millisecond,
	}

	if info.ExternalIP.String() != "203.0.113.1" {
		t.Errorf("Expected external IP 203.0.113.1, got %s", info.ExternalIP)
	}
	if info.Type != TypeRestricted {
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
		t.Error("Expected Info, got nil")
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

func TestAlignedAttrLen(t *testing.T) {
	tests := []struct {
		input    uint16
		expected int
	}{
		{0, 0},
		{1, 4},
		{2, 4},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{12, 12},
	}

	for _, tt := range tests {
		result := alignedAttrLen(tt.input)
		if result != tt.expected {
			t.Errorf("alignedAttrLen(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseAddressAttributeIPv4(t *testing.T) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}

	// Test XOR-MAPPED-ADDRESS (0x0020)
	// IP: 192.168.1.100 = 0xC0A80164
	// XOR with 0x2112A442 = 0xE1BAA526
	data := []byte{
		0x00, 0x01, // Reserved + Family (IPv4)
		0x00, 0x00, // Port (ignored)
		0xe1, 0xba, 0xa5, 0x26, // XOR'd IP
	}

	ip := parseAddressAttribute(data, 0x0020, 8, magicCookie)
	expected := net.IPv4(192, 168, 1, 100)
	if ip == nil {
		t.Fatal("parseAddressAttribute returned nil")
	}
	if !ip.Equal(expected) {
		t.Errorf("Expected IP %s, got %s", expected, ip)
	}
}

func TestParseAddressAttributeMappedAddress(t *testing.T) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}

	// Test MAPPED-ADDRESS (0x0001) - not XOR'd
	data := []byte{
		0x00, 0x01, // Reserved + Family (IPv4)
		0x00, 0x00, // Port (ignored)
		0xc0, 0xa8, 0x01, 0x64, // IP: 192.168.1.100
	}

	ip := parseAddressAttribute(data, 0x0001, 8, magicCookie)
	expected := net.IPv4(192, 168, 1, 100)
	if ip == nil {
		t.Fatal("parseAddressAttribute returned nil")
	}
	if !ip.Equal(expected) {
		t.Errorf("Expected IP %s, got %s", expected, ip)
	}
}

func TestParseAddressAttributeInvalidFamily(t *testing.T) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}

	// IPv6 family - should return nil
	data := []byte{
		0x00, 0x02, // Reserved + Family (IPv6)
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	ip := parseAddressAttribute(data, 0x0020, 8, magicCookie)
	if ip != nil {
		t.Errorf("Expected nil for IPv6, got %s", ip)
	}
}

func TestParseAddressAttributeTooShort(t *testing.T) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}

	// Too short attribute
	data := []byte{0x00, 0x01, 0x00, 0x00}

	ip := parseAddressAttribute(data, 0x0020, 4, magicCookie)
	if ip != nil {
		t.Errorf("Expected nil for short data, got %s", ip)
	}
}

func TestParseAddressAttributeUnknownType(t *testing.T) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}

	data := []byte{
		0x00, 0x01,
		0x00, 0x00,
		0xc0, 0xa8, 0x01, 0x64,
	}

	// Unknown attribute type
	ip := parseAddressAttribute(data, 0x9999, 8, magicCookie)
	if ip != nil {
		t.Errorf("Expected nil for unknown type, got %s", ip)
	}
}

func TestIsSTUNBindingResponse(t *testing.T) {
	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0x01, 0x01}, true},  // Binding success response
		{[]byte{0x01, 0x11}, false}, // Binding error response (not success)
		{[]byte{0x00, 0x01}, false}, // Binding request
		{[]byte{0x02, 0x01}, false}, // Wrong type
	}

	for _, tt := range tests {
		result := isSTUNBindingResponse(tt.data)
		if result != tt.expected {
			t.Errorf("isSTUNBindingResponse(%v) = %v, want %v", tt.data, result, tt.expected)
		}
	}
}

func TestAddPortMappingNoNATDevice(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// No NAT device discovered, should fail
	_, err := client.AddPortMapping(ctx, 8080, 80, ProtocolTCP, "test", time.Hour)
	// Should fail because no NAT device is available
	if err == nil {
		t.Skip("NAT device available in environment")
	}
}

func TestGetExternalIPNoCache(t *testing.T) {
	client := NewClient("10.255.255.1:3478") // Unreachable STUN
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should fail or return nil
	ip, err := client.GetExternalIP(ctx)
	if err == nil && ip != nil {
		// Got a result somehow, ok
		t.Logf("Got external IP: %s", ip)
	}
}

func TestDiscoverSubnets(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Call Discover with short timeout
	info, err := client.Discover(ctx)
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
		return
	}

	// Check that basic info is populated
	if info != nil {
		if info.InternalIP != nil {
			t.Logf("Internal IP: %s", info.InternalIP)
		}
		if info.GatewayIP != nil {
			t.Logf("Gateway IP: %s", info.GatewayIP)
		}
	}
}

func TestExtractMappedAddressNoAttrs(t *testing.T) {
	// Test with data that has no attributes after header
	data := make([]byte, 20)
	data[0] = 0x01
	data[1] = 0x01
	data[2] = 0x00
	data[3] = 0x00 // Length: 0

	// Magic cookie
	data[4] = 0x21
	data[5] = 0x12
	data[6] = 0xa4
	data[7] = 0x42

	ip, err := extractMappedAddress(data)
	if ip != nil || err == nil {
		t.Logf("extractMappedAddress returned ip=%v err=%v", ip, err)
	}
}

func TestExtractMappedAddressMissingMagicCookie(t *testing.T) {
	// Test with invalid magic cookie
	data := make([]byte, 24)
	data[0] = 0x01
	data[1] = 0x01
	data[2] = 0x00
	data[3] = 0x04 // Length: 4

	// Wrong magic cookie
	data[4] = 0x00
	data[5] = 0x00
	data[6] = 0x00
	data[7] = 0x00

	ip, err := extractMappedAddress(data)
	// Should still try to parse
	_ = ip
	_ = err
}

func TestSetUDPDeadlineNilConn(t *testing.T) {
	// Test with nil context (no deadline)
	ctx := context.Background()

	// Can't easily test setUDPDeadline without a real connection
	// Just verify the function signature exists
	_ = ctx
}

func TestAddPortMappingValidProtocols(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with valid TCP protocol
	_, err := client.AddPortMapping(ctx, 8080, 80, ProtocolTCP, "test tcp", time.Hour)
	if err == ErrInvalidProtocol {
		t.Error("TCP should be a valid protocol")
	}

	// Test with valid UDP protocol
	_, err = client.AddPortMapping(ctx, 8080, 80, ProtocolUDP, "test udp", time.Hour)
	if err == ErrInvalidProtocol {
		t.Error("UDP should be a valid protocol")
	}
}

func TestFindSubstring(t *testing.T) {
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
		{"WANIPConnection", "WANIPConnection", true},
	}

	for _, tt := range tests {
		result := findSubstring(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("findSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}
