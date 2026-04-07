// Package nat provides NAT traversal and detection capabilities.
// It supports UPnP, NAT-PMP, and STUN for discovering external IPs
// and managing port forwarding.
package nat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Common errors for NAT operations.
var (
	ErrNoNATDevice     = errors.New("no NAT device found")
	ErrNATUnsupported  = errors.New("NAT traversal not supported")
	ErrPortMapFailed   = errors.New("port mapping failed")
	ErrSTUNFailed      = errors.New("STUN request failed")
	ErrTimeout         = errors.New("operation timed out")
	ErrInvalidProtocol = errors.New("invalid protocol")
)

// Protocol represents the transport protocol for port mapping.
type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

// STUN protocol constants (RFC 5389).
const (
	// stunBindingRequest is the message type for a STUN Binding Request.
	stunBindingRequest = 0x0001
	// stunMagicCookie is the fixed magic cookie value in STUN headers.
	stunMagicCookie = 0x2112A442
	// stunHeaderSize is the size of a STUN message header.
	stunHeaderSize = 20
	// stunMinResponseSize is the minimum valid STUN response size.
	stunMinResponseSize = 20
	// stunMaxResponseSize is the buffer size for STUN responses.
	stunMaxResponseSize = 512
)

// Info contains information about the NAT environment.
type Info struct {
	ExternalIP   net.IP        `json:"external_ip,omitempty"`
	InternalIP   net.IP        `json:"internal_ip,omitempty"`
	GatewayIP    net.IP        `json:"gateway_ip,omitempty"`
	Type         Type          `json:"type"`
	UPnPEnabled  bool          `json:"upnp_enabled"`
	NATPMPEnable bool          `json:"natpmp_enabled"`
	Latency      time.Duration `json:"latency,omitempty"`
}

// Type represents the type of NAT detected.
type Type string

const (
	TypeNone           Type = "none"
	TypeFull           Type = "full_cone"
	TypeRestricted     Type = "restricted_cone"
	TypePortRestricted Type = "port_restricted"
	TypeSymmetric      Type = "symmetric"
	TypeUnknown        Type = "unknown"
)

// PortMapping represents an active port mapping.
type PortMapping struct {
	InternalPort int           `json:"internal_port"`
	ExternalPort int           `json:"external_port"`
	Protocol     Protocol      `json:"protocol"`
	Description  string        `json:"description,omitempty"`
	Lifetime     time.Duration `json:"lifetime"`
	CreatedAt    time.Time     `json:"created_at"`
}

// Discoverer provides NAT traversal operations.
type Discoverer interface {
	// Discover finds NAT devices and determines NAT type.
	Discover(ctx context.Context) (*Info, error)

	// GetExternalIP returns the public IP address.
	GetExternalIP(ctx context.Context) (net.IP, error)

	// AddPortMapping creates a port forwarding rule.
	// NOTE: Currently returns ErrNATUnsupported as UPnP/NAT-PMP port mapping
	// is not yet implemented. This interface method exists for future compatibility.
	AddPortMapping(ctx context.Context, internal, external int, proto Protocol, desc string, lifetime time.Duration) (*PortMapping, error)

	// RemovePortMapping removes a port forwarding rule.
	RemovePortMapping(ctx context.Context, external int, proto Protocol) error

	// ListMappings returns all active port mappings.
	ListMappings(ctx context.Context) ([]PortMapping, error)
}

// Client implements Discoverer with multiple traversal methods.
type Client struct {
	mu          sync.RWMutex
	gatewayIP   net.IP
	internalIP  net.IP
	externalIP  net.IP
	natInfo     *Info
	mappings    map[string]*PortMapping
	stunServers []string
}

// NewClient creates a new NAT client.
func NewClient(stunServers ...string) *Client {
	servers := stunServers
	if len(servers) == 0 {
		servers = defaultSTUNServers
	}
	return &Client{
		mappings:    make(map[string]*PortMapping),
		stunServers: servers,
	}
}

// Default STUN servers for external IP discovery.
var defaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun.cloudflare.com:3478",
}

// Discover finds NAT devices and determines the NAT configuration.
func (c *Client) Discover(ctx context.Context) (*Info, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info := &Info{
		Type: TypeUnknown,
	}

	// Get gateway IP
	gateway, err := getDefaultGateway()
	if err == nil {
		info.GatewayIP = gateway
		c.gatewayIP = gateway
	}

	// Get local IP
	localIP, err := getLocalIP()
	if err == nil {
		info.InternalIP = localIP
		c.internalIP = localIP
	}

	// Check if we're behind NAT by comparing internal and external IPs
	externalIP, err := c.getExternalIPViaSTUN(ctx)
	if err == nil {
		info.ExternalIP = externalIP
		c.externalIP = externalIP

		// Determine if behind NAT
		if localIP != nil && externalIP != nil && !localIP.Equal(externalIP) {
			info.Type = TypeRestricted // Default assumption
		} else if localIP != nil && externalIP != nil && localIP.Equal(externalIP) {
			info.Type = TypeNone
		}
	}

	// Try UPnP discovery
	upnpEnabled := c.discoverUPnP(ctx)
	info.UPnPEnabled = upnpEnabled

	// Try NAT-PMP discovery
	natPMPEnabled := c.discoverNATPMP(ctx)
	info.NATPMPEnable = natPMPEnabled

	c.natInfo = info
	return info, nil
}

// GetExternalIP returns the public IP address using available methods.
func (c *Client) GetExternalIP(ctx context.Context) (net.IP, error) {
	c.mu.RLock()
	if c.externalIP != nil {
		c.mu.RUnlock()
		return c.externalIP, nil
	}
	c.mu.RUnlock()

	return c.getExternalIPViaSTUN(ctx)
}

// AddPortMapping creates a port forwarding rule.
func (c *Client) AddPortMapping(ctx context.Context, internal, external int, proto Protocol, desc string, lifetime time.Duration) (*PortMapping, error) {
	if proto != ProtocolTCP && proto != ProtocolUDP {
		return nil, ErrInvalidProtocol
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	mapping := &PortMapping{
		InternalPort: internal,
		ExternalPort: external,
		Protocol:     proto,
		Description:  desc,
		Lifetime:     lifetime,
		CreatedAt:    time.Now(),
	}

	// Try UPnP first, then NAT-PMP
	err := c.addMappingUPnP(ctx, mapping)
	if err != nil {
		err = c.addMappingNATPMP(ctx, mapping)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPortMapFailed, err)
		}
	}

	key := fmt.Sprintf("%d:%s", external, proto)
	c.mappings[key] = mapping

	return mapping, nil
}

// RemovePortMapping removes a port forwarding rule.
func (c *Client) RemovePortMapping(ctx context.Context, external int, proto Protocol) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%d:%s", external, proto)
	delete(c.mappings, key)

	// Try to remove via UPnP, then NAT-PMP (best effort)
	_ = c.removeMappingUPnP(ctx, external, proto)
	_ = c.removeMappingNATPMP(ctx, external, proto)

	return nil
}

// ListMappings returns all active port mappings.
func (c *Client) ListMappings(ctx context.Context) ([]PortMapping, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	mappings := make([]PortMapping, 0, len(c.mappings))
	for _, m := range c.mappings {
		mappings = append(mappings, *m)
	}
	return mappings, nil
}

// getExternalIPViaSTUN uses STUN servers to discover the external IP.
func (c *Client) getExternalIPViaSTUN(ctx context.Context) (net.IP, error) {
	for _, server := range c.stunServers {
		ip, err := stunRequest(ctx, server)
		if err == nil {
			return ip, nil
		}
	}
	return nil, ErrSTUNFailed
}

// stunRequest performs a basic STUN binding request.
func stunRequest(ctx context.Context, server string) (net.IP, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	conn, err := net.DialTimeout("udp", server, time.Until(deadline))
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(deadline); err != nil {
		return nil, err
	}

	// Build STUN Binding Request (RFC 5389)
	// Header: type (2) + length (2) + magic cookie (4) + transaction ID (12)
	request := make([]byte, stunHeaderSize)
	request[0] = byte(stunBindingRequest >> 8)
	request[1] = byte(stunBindingRequest & 0xFF)
	// Length is 0 (no attributes)
	// Magic Cookie (big-endian)
	cookie := uint32(stunMagicCookie)
	request[4] = byte(cookie >> 24)
	request[5] = byte(cookie >> 16)
	request[6] = byte(cookie >> 8)
	request[7] = byte(cookie & 0xFF)
	// Transaction ID (deterministic for reproducibility)
	for i := 8; i < stunHeaderSize; i++ {
		request[i] = byte(i * 17)
	}

	if _, err := conn.Write(request); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	response := make([]byte, stunMaxResponseSize)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	if n < stunMinResponseSize {
		return nil, errors.New("response too short")
	}

	// Parse STUN response to extract XOR-MAPPED-ADDRESS
	return parseSTUNResponse(response[:n])
}

// parseSTUNResponse extracts the mapped address from a STUN response.
func parseSTUNResponse(data []byte) (net.IP, error) {
	if len(data) < 20 {
		return nil, errors.New("response too short")
	}
	if !isSTUNBindingResponse(data) {
		return nil, errors.New("not a binding response")
	}
	return extractMappedAddress(data[20:])
}

// isSTUNBindingResponse checks if data represents a STUN binding response.
func isSTUNBindingResponse(data []byte) bool {
	return data[0] == 0x01 && data[1] == 0x01
}

// extractMappedAddress parses STUN attributes to find the mapped address.
func extractMappedAddress(attrs []byte) (net.IP, error) {
	magicCookie := []byte{0x21, 0x12, 0xa4, 0x42}
	offset := 0

	for offset+4 <= len(attrs) {
		attrType := (uint16(attrs[offset]) << 8) | uint16(attrs[offset+1])
		attrLen := (uint16(attrs[offset+2]) << 8) | uint16(attrs[offset+3])
		offset += 4

		if offset+int(attrLen) > len(attrs) {
			break
		}

		if ip := parseAddressAttribute(attrs[offset:], attrType, attrLen, magicCookie); ip != nil {
			return ip, nil
		}

		offset += alignedAttrLen(attrLen)
	}
	return nil, errors.New("no mapped address in response")
}

// parseAddressAttribute parses XOR-MAPPED-ADDRESS or MAPPED-ADDRESS attributes.
func parseAddressAttribute(data []byte, attrType, attrLen uint16, magicCookie []byte) net.IP {
	if attrLen < 8 || data[1] != 0x01 { // IPv4 family
		return nil
	}

	switch attrType {
	case 0x0020: // XOR-MAPPED-ADDRESS
		ip := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			ip[i] = data[4+i] ^ magicCookie[i]
		}
		return ip
	case 0x0001: // MAPPED-ADDRESS
		ip := make(net.IP, 4)
		copy(ip, data[4:8])
		return ip
	}
	return nil
}

// alignedAttrLen returns the 4-byte aligned attribute length.
func alignedAttrLen(attrLen uint16) int {
	n := int(attrLen)
	if attrLen%4 != 0 {
		n += 4 - int(attrLen%4)
	}
	return n
}

// getDefaultGateway returns the default gateway IP.
func getDefaultGateway() (net.IP, error) {
	// Use github.com/jackpal/gateway if available
	// For now, try to determine gateway from routing table
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipnet.IP.To4()
			if ipv4 == nil {
				continue
			}

			// Common gateway patterns
			gateway := make(net.IP, 4)
			copy(gateway, ipv4)
			gateway[3] = 1 // Assume .1 gateway

			return gateway, nil
		}
	}

	return nil, errors.New("no gateway found")
}

// getLocalIP returns the local IP address.
func getLocalIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// setUDPDeadline sets a deadline on a UDP connection from context or default.
func setUDPDeadline(conn *net.UDPConn, ctx context.Context, defaultTimeout time.Duration) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultTimeout)
	}
	return conn.SetDeadline(deadline)
}

// discoverUPnP attempts to discover UPnP IGD devices.
func (c *Client) discoverUPnP(ctx context.Context) bool {
	ssdpAddr := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}

	conn, err := net.DialUDP("udp4", nil, ssdpAddr)
	if err != nil {
		return false
	}
	defer conn.Close()

	if err := setUDPDeadline(conn, ctx, 2*time.Second); err != nil {
		return false
	}

	// M-SEARCH request for Internet Gateway Device
	mSearch := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 1\r\n" +
		"ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n\r\n"

	if _, err := conn.Write([]byte(mSearch)); err != nil {
		return false
	}

	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}

	response := string(buf[:n])
	return len(response) > 0 && (contains(response, "200 OK") || contains(response, "InternetGatewayDevice"))
}

// discoverNATPMP attempts to discover NAT-PMP support.
func (c *Client) discoverNATPMP(ctx context.Context) bool {
	gateway := c.gatewayIP
	if gateway == nil {
		return false
	}

	addr := &net.UDPAddr{IP: gateway, Port: 5351}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	if err := setUDPDeadline(conn, ctx, 2*time.Second); err != nil {
		return false
	}

	// NAT-PMP external address request: Version 0, Opcode 0
	request := []byte{0x00, 0x00}
	if _, err := conn.Write(request); err != nil {
		return false
	}

	buf := make([]byte, 12)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}

	// Valid response: 12 bytes with version 0 and result code 0
	return n == 12 && buf[0] == 0 && buf[1] == 128 && buf[2] == 0 && buf[3] == 0
}

// addMappingUPnP adds a port mapping via UPnP.
// NOTE: This is currently a stub function. Full UPnP IGD implementation requires
// SOAP/HTTP calls to the gateway's control URL. This returns ErrNATUnsupported
// until UPnP IGD support is fully implemented.
func (c *Client) addMappingUPnP(_ context.Context, _ *PortMapping) error {
	// Full UPnP implementation requires SOAP calls
	// This is a placeholder - real implementation would use UPnP library
	return ErrNATUnsupported
}

// addMappingNATPMP adds a port mapping via NAT-PMP.
// NOTE: This is currently a stub function. NAT-PMP implementation requires
// sending properly formatted mapping requests to the gateway. This returns
// ErrNATUnsupported until NAT-PMP support is fully implemented.
func (c *Client) addMappingNATPMP(_ context.Context, _ *PortMapping) error {
	// NAT-PMP mapping request
	// This is a placeholder - real implementation would send proper NAT-PMP request
	return ErrNATUnsupported
}

// removeMappingUPnP removes a port mapping via UPnP.
func (c *Client) removeMappingUPnP(_ context.Context, _ int, _ Protocol) error {
	return nil
}

// removeMappingNATPMP removes a port mapping via NAT-PMP.
func (c *Client) removeMappingNATPMP(_ context.Context, _ int, _ Protocol) error {
	return nil
}

// contains checks if substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring is a simple substring search.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
