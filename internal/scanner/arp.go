// Package scanner provides network scanning functionality for device discovery.
package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// ARPScanner implements ARP-based network scanning for device discovery.
// It uses raw sockets via gopacket/pcap for fast layer 2 discovery.
type ARPScanner struct {
	iface     *net.Interface
	workers   int
	timeout   time.Duration
	retries   int
	localIP   net.IP
	localMAC  net.HardwareAddr
	ouiDB     OUIDatabase
	ouiDBOnce sync.Once
}

// OUIDatabase provides MAC vendor lookup functionality.
type OUIDatabase interface {
	// Lookup returns the vendor name for a MAC address prefix.
	Lookup(mac net.HardwareAddr) string
}

// NewARPScanner creates a new ARP scanner for the given interface.
func NewARPScanner(ifaceName string, workers int, timeout time.Duration, retries int) (*ARPScanner, error) {
	var iface *net.Interface
	var err error

	if ifaceName == "" {
		iface, err = getDefaultInterface()
	} else {
		iface, err = net.InterfaceByName(ifaceName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	localIP, err := getInterfaceIP(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface IP: %w", err)
	}

	return &ARPScanner{
		iface:    iface,
		workers:  workers,
		timeout:  timeout,
		retries:  retries,
		localIP:  localIP,
		localMAC: iface.HardwareAddr,
	}, nil
}

// Name returns the scanner name.
func (s *ARPScanner) Name() string {
	return "arp"
}

// SetOUIDatabase sets the OUI database for vendor lookups.
func (s *ARPScanner) SetOUIDatabase(db OUIDatabase) {
	s.ouiDB = db
}

// Scan performs an ARP scan on the given subnet.
func (s *ARPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	handle, err := s.openPcapHandle()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	ips := generateIPs(ipNet)
	if len(ips) == 0 {
		return nil, nil
	}

	results := make(chan Device, len(ips))
	seen := &sync.Map{}

	listenerCtx, cancelListener := context.WithCancel(ctx)
	var listenerWg sync.WaitGroup
	listenerWg.Add(1)
	go func() {
		defer listenerWg.Done()
		s.listenForResponses(listenerCtx, handle, results, seen)
	}()

	s.sendARPRequests(ctx, handle, ips)

	// Give time for final responses
	select {
	case <-time.After(s.timeout):
	case <-ctx.Done():
	}

	cancelListener()
	listenerWg.Wait()
	close(results)

	return collectDevices(results), nil
}

// openPcapHandle opens a pcap handle with ARP BPF filter.
func (s *ARPScanner) openPcapHandle() (*pcap.Handle, error) {
	handle, err := pcap.OpenLive(s.iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open pcap handle: %w", err)
	}
	if err := handle.SetBPFFilter("arp"); err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to set BPF filter: %w", err)
	}
	return handle, nil
}

// sendARPRequests distributes IPs to worker goroutines for ARP request sending.
func (s *ARPScanner) sendARPRequests(ctx context.Context, handle *pcap.Handle, ips []net.IP) {
	var wg sync.WaitGroup
	ipChan := make(chan net.IP, len(ips))

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.arpWorker(ctx, handle, ipChan, s.retries)
		}()
	}

	go func() {
		defer close(ipChan)
		for _, ip := range ips {
			select {
			case ipChan <- ip:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
}

// collectDevices drains a device channel into a slice.
func collectDevices(ch <-chan Device) []Device {
	var devices []Device
	for device := range ch {
		devices = append(devices, device)
	}
	return devices
}

// arpWorker sends ARP requests for IPs from the channel.
func (s *ARPScanner) arpWorker(ctx context.Context, handle *pcap.Handle, ips <-chan net.IP, retries int) {
	for ip := range ips {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for i := 0; i <= retries; i++ {
			if err := s.sendARPRequest(handle, ip); err != nil {
				continue
			}
			time.Sleep(10 * time.Millisecond) // Small delay between retries
		}
	}
}

// sendARPRequest sends an ARP request for the target IP.
func (s *ARPScanner) sendARPRequest(handle *pcap.Handle, targetIP net.IP) error {
	// Construct Ethernet frame
	eth := layers.Ethernet{
		SrcMAC:       s.localMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	// Construct ARP request
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(s.localMAC),
		SourceProtAddress: []byte(s.localIP.To4()),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(targetIP.To4()),
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
		return fmt.Errorf("failed to serialize ARP packet: %w", err)
	}

	return handle.WritePacketData(buf.Bytes())
}

// listenForResponses captures ARP responses and adds devices to results.
func (s *ARPScanner) listenForResponses(ctx context.Context, handle *pcap.Handle, results chan<- Device, seen *sync.Map) {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.NoCopy = true

	for {
		select {
		case <-ctx.Done():
			return
		case packet := <-packetSource.Packets():
			if device, ok := s.processARPPacket(packet, seen); ok {
				select {
				case results <- device:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// processARPPacket extracts device info from an ARP reply packet.
// Returns the device and true if successful, zero Device and false otherwise.
func (s *ARPScanner) processARPPacket(packet gopacket.Packet, seen *sync.Map) (Device, bool) {
	if packet == nil {
		return Device{}, false
	}

	arp, ok := s.extractARPReply(packet)
	if !ok {
		return Device{}, false
	}

	ip := net.IP(arp.SourceProtAddress)
	if _, loaded := seen.LoadOrStore(ip.String(), true); loaded {
		return Device{}, false
	}

	return s.createDevice(ip, net.HardwareAddr(arp.SourceHwAddress)), true
}

// extractARPReply extracts ARP reply layer from a packet.
func (s *ARPScanner) extractARPReply(packet gopacket.Packet) (*layers.ARP, bool) {
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		return nil, false
	}
	arp := arpLayer.(*layers.ARP)
	if arp.Operation != layers.ARPReply {
		return nil, false
	}
	return arp, true
}

// createDevice creates a Device from IP and MAC address.
func (s *ARPScanner) createDevice(ip net.IP, mac net.HardwareAddr) Device {
	now := time.Now()
	device := Device{
		IP:        ip,
		MAC:       mac,
		LastSeen:  now,
		FirstSeen: now,
		Status:    StatusNew,
		Metadata:  make(map[string]interface{}),
	}
	if s.ouiDB != nil {
		device.Vendor = s.ouiDB.Lookup(mac)
	}
	return device
}

// getDefaultInterface returns the default network interface.
func getDefaultInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if interface has an IPv4 address
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				return &iface, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable network interface found")
}

// getInterfaceIP returns the IPv4 address of the interface.
func getInterfaceIP(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			return ipNet.IP.To4(), nil
		}
	}

	return nil, fmt.Errorf("no IPv4 address found on interface %s", iface.Name)
}

// generateIPs generates all IP addresses in a subnet (excluding network and broadcast).
func generateIPs(ipNet *net.IPNet) []net.IP {
	var ips []net.IP

	// Get subnet mask size
	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones

	// Calculate number of hosts (excluding network and broadcast)
	numHosts := (1 << hostBits) - 2
	if numHosts <= 0 {
		return nil
	}

	// Start from first host address
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)

	// Increment to first host
	inc(ip)

	for i := 0; i < numHosts; i++ {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		ips = append(ips, newIP)
		inc(ip)
	}

	return ips
}

// inc increments an IP address by 1.
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
