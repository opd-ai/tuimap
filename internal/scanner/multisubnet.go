// Package scanner provides network scanning functionality for device discovery.
package scanner

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SubnetInfo contains information about a discovered subnet.
type SubnetInfo struct {
	Subnet    string // CIDR notation
	Gateway   net.IP // Gateway IP if known
	Interface string // Network interface name
	Local     bool   // Whether this is a local subnet
}

// MultiSubnetScanner discovers and scans multiple subnets.
type MultiSubnetScanner struct {
	orchestrator *Orchestrator
}

// NewMultiSubnetScanner creates a scanner that can handle multiple subnets.
func NewMultiSubnetScanner(orch *Orchestrator) *MultiSubnetScanner {
	return &MultiSubnetScanner{orchestrator: orch}
}

// DiscoverSubnets finds all local subnets on the system.
func DiscoverSubnets() ([]SubnetInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var subnets []SubnetInfo
	for _, iface := range ifaces {
		if !isUsableInterface(iface) {
			continue
		}
		subnets = append(subnets, discoverInterfaceSubnets(iface)...)
	}
	return subnets, nil
}

// isUsableInterface checks if an interface is suitable for scanning.
func isUsableInterface(iface net.Interface) bool {
	return iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0
}

// discoverInterfaceSubnets extracts subnet info from an interface.
func discoverInterfaceSubnets(iface net.Interface) []SubnetInfo {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}

	var subnets []SubnetInfo
	for _, addr := range addrs {
		if subnet, ok := parseSubnetFromAddr(addr, iface.Name); ok {
			subnets = append(subnets, subnet)
		}
	}
	return subnets
}

// parseSubnetFromAddr extracts SubnetInfo from a network address.
func parseSubnetFromAddr(addr net.Addr, ifaceName string) (SubnetInfo, bool) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		return SubnetInfo{}, false
	}

	ipv4 := ipNet.IP.To4()
	if ipv4 == nil || ipv4.IsLinkLocalUnicast() {
		return SubnetInfo{}, false
	}

	ones, bits := ipNet.Mask.Size()
	if ones == 0 || bits == 0 {
		return SubnetInfo{}, false
	}

	networkIP := ipv4.Mask(ipNet.Mask)
	gateway := make(net.IP, 4)
	copy(gateway, networkIP)
	gateway[3] = 1

	return SubnetInfo{
		Subnet:    fmt.Sprintf("%s/%d", networkIP, ones),
		Gateway:   gateway,
		Interface: ifaceName,
		Local:     true,
	}, true
}

// ParseRoutingTable reads the system routing table and extracts subnets.
func ParseRoutingTable() ([]SubnetInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return parseLinuxRoutingTable()
	case "darwin":
		return parseDarwinRoutingTable()
	default:
		// Fall back to interface-based discovery
		return DiscoverSubnets()
	}
}

// parseLinuxRoutingTable parses /proc/net/route on Linux.
func parseLinuxRoutingTable() ([]SubnetInfo, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return DiscoverSubnets()
	}
	defer file.Close()

	var subnets []SubnetInfo
	scanner := bufio.NewScanner(file)

	// Skip header line
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		iface := fields[0]
		destHex := fields[1]
		gwHex := fields[2]
		maskHex := fields[7]

		// Parse hex values (little-endian on x86)
		dest := parseHexIP(destHex)
		gateway := parseHexIP(gwHex)
		mask := parseHexIP(maskHex)

		if dest == nil || mask == nil {
			continue
		}

		// Skip default route (0.0.0.0/0)
		if dest.Equal(net.IPv4zero) && mask.Equal(net.IPv4zero) {
			continue
		}

		// Calculate CIDR prefix
		ones, _ := net.IPMask(mask).Size()
		if ones == 0 {
			continue
		}

		subnet := fmt.Sprintf("%s/%d", dest, ones)
		subnets = append(subnets, SubnetInfo{
			Subnet:    subnet,
			Gateway:   gateway,
			Interface: iface,
			Local:     gateway == nil || gateway.Equal(net.IPv4zero),
		})
	}

	if len(subnets) == 0 {
		return DiscoverSubnets()
	}

	return subnets, nil
}

// parseHexIP parses a hex-encoded IP address from /proc/net/route.
func parseHexIP(hex string) net.IP {
	if len(hex) != 8 {
		return nil
	}

	// Parse as little-endian (Linux x86 format)
	ip := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		var b byte
		_, err := fmt.Sscanf(hex[6-2*i:8-2*i], "%02x", &b)
		if err != nil {
			return nil
		}
		ip[i] = b
	}
	return ip
}

// parseDarwinRoutingTable uses netstat on macOS.
func parseDarwinRoutingTable() ([]SubnetInfo, error) {
	// On macOS, fall back to interface discovery
	// Full implementation would parse 'netstat -rn' output
	return DiscoverSubnets()
}

// DeduplicateSubnets removes duplicate subnets from a list.
func DeduplicateSubnets(subnets []SubnetInfo) []SubnetInfo {
	seen := make(map[string]bool)
	var result []SubnetInfo

	for _, subnet := range subnets {
		if !seen[subnet.Subnet] {
			seen[subnet.Subnet] = true
			result = append(result, subnet)
		}
	}

	return result
}

// FilterLocalSubnets returns only local subnets (directly connected).
func FilterLocalSubnets(subnets []SubnetInfo) []SubnetInfo {
	var result []SubnetInfo
	for _, subnet := range subnets {
		if subnet.Local {
			result = append(result, subnet)
		}
	}
	return result
}

// GetDefaultSubnet returns the most likely "primary" subnet to scan.
func GetDefaultSubnet() (string, error) {
	subnets, err := DiscoverSubnets()
	if err != nil {
		return "", err
	}

	if len(subnets) == 0 {
		return "", fmt.Errorf("no suitable subnet found")
	}

	// Prefer non-link-local, private subnets
	for _, subnet := range subnets {
		_, ipNet, _ := net.ParseCIDR(subnet.Subnet)
		if ipNet == nil {
			continue
		}

		ip := ipNet.IP
		// Prefer common private ranges
		if ip[0] == 192 && ip[1] == 168 {
			return subnet.Subnet, nil
		}
		if ip[0] == 10 {
			return subnet.Subnet, nil
		}
		if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
			return subnet.Subnet, nil
		}
	}

	// Return first subnet if no private subnet found
	return subnets[0].Subnet, nil
}

// MultiSubnetScanResult contains results from scanning multiple subnets.
type MultiSubnetScanResult struct {
	Results    map[string]*ScanResult // Results by subnet CIDR
	AllDevices []Device               // Merged and deduplicated devices
	TotalTime  time.Duration          // Total scan duration
	Subnets    []SubnetInfo           // Subnets that were scanned
}

// ScanAllSubnets discovers all local subnets and scans them in parallel.
func (m *MultiSubnetScanner) ScanAllSubnets(ctx context.Context) (*MultiSubnetScanResult, error) {
	subnets, err := DiscoverSubnets()
	if err != nil {
		return nil, fmt.Errorf("failed to discover subnets: %w", err)
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets discovered")
	}

	// Deduplicate subnets
	subnets = DeduplicateSubnets(subnets)

	return m.ScanSubnets(ctx, subnets)
}

// ScanSubnets scans the given subnets in parallel and merges results.
func (m *MultiSubnetScanner) ScanSubnets(ctx context.Context, subnets []SubnetInfo) (*MultiSubnetScanResult, error) {
	startTime := time.Now()

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets to scan")
	}

	// Result collection
	type subnetResult struct {
		subnet string
		result *ScanResult
		err    error
	}

	resultChan := make(chan subnetResult, len(subnets))
	var wg sync.WaitGroup

	// Scan each subnet in parallel
	for _, subnet := range subnets {
		wg.Add(1)
		go func(s SubnetInfo) {
			defer wg.Done()
			result, err := m.orchestrator.Scan(ctx, s.Subnet)
			resultChan <- subnetResult{
				subnet: s.Subnet,
				result: result,
				err:    err,
			}
		}(subnet)
	}

	// Close channel when all scans complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make(map[string]*ScanResult)
	deviceMap := make(map[string]*Device) // Deduplicate devices by MAC or IP

	for sr := range resultChan {
		if sr.err != nil {
			// Log error but continue with other subnets
			continue
		}
		results[sr.subnet] = sr.result

		// Merge devices into global map
		for _, device := range sr.result.Devices {
			key := deviceKey(&device)
			if existing, exists := deviceMap[key]; exists {
				mergeDevices(existing, &device)
			} else {
				deviceCopy := device
				deviceMap[key] = &deviceCopy
			}
		}
	}

	// Convert device map to slice
	allDevices := make([]Device, 0, len(deviceMap))
	for _, device := range deviceMap {
		allDevices = append(allDevices, *device)
	}

	return &MultiSubnetScanResult{
		Results:    results,
		AllDevices: allDevices,
		TotalTime:  time.Since(startTime),
		Subnets:    subnets,
	}, nil
}

// ScanLocalSubnets discovers and scans only local (directly connected) subnets.
func (m *MultiSubnetScanner) ScanLocalSubnets(ctx context.Context) (*MultiSubnetScanResult, error) {
	subnets, err := DiscoverSubnets()
	if err != nil {
		return nil, fmt.Errorf("failed to discover subnets: %w", err)
	}

	// Filter to local subnets only
	localSubnets := FilterLocalSubnets(subnets)
	if len(localSubnets) == 0 {
		return nil, fmt.Errorf("no local subnets found")
	}

	// Deduplicate
	localSubnets = DeduplicateSubnets(localSubnets)

	return m.ScanSubnets(ctx, localSubnets)
}

// ScanFromRoutingTable parses the system routing table and scans all subnets.
func (m *MultiSubnetScanner) ScanFromRoutingTable(ctx context.Context) (*MultiSubnetScanResult, error) {
	subnets, err := ParseRoutingTable()
	if err != nil {
		return nil, fmt.Errorf("failed to parse routing table: %w", err)
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets found in routing table")
	}

	// Deduplicate
	subnets = DeduplicateSubnets(subnets)

	return m.ScanSubnets(ctx, subnets)
}
