// Package scanner provides network scanning functionality for device discovery.

//go:build linux

package scanner

import (
"context"
"fmt"
"net"
"sync"
"time"

"github.com/gopacket/gopacket"
"github.com/gopacket/gopacket/afpacket"
"github.com/gopacket/gopacket/layers"
"golang.org/x/net/bpf"
)

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

// Scan performs an ARP scan on the given subnet.
func (s *ARPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
_, ipNet, err := net.ParseCIDR(subnet)
if err != nil {
return nil, fmt.Errorf("invalid subnet: %w", err)
}

handle, err := s.openHandle()
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

// arpBPFFilter returns compiled BPF instructions that match ARP packets (EtherType 0x0806).
func arpBPFFilter() ([]bpf.RawInstruction, error) {
// Filter: match Ethernet frames with EtherType == ARP (0x0806)
return bpf.Assemble([]bpf.Instruction{
bpf.LoadAbsolute{Off: 12, Size: 2},                                        // load EtherType at offset 12
bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0x0806, SkipTrue: 1, SkipFalse: 0}, // if ARP, skip to accept
bpf.RetConstant{Val: 0},                                                   // reject
bpf.RetConstant{Val: 0xffff},                                              // accept
})
}

// openHandle opens an AF_PACKET handle with ARP BPF filter.
func (s *ARPScanner) openHandle() (*afpacket.TPacket, error) {
handle, err := afpacket.NewTPacket(afpacket.OptInterface(s.iface.Name))
if err != nil {
return nil, fmt.Errorf("failed to open AF_PACKET handle: %w", err)
}
filter, err := arpBPFFilter()
if err != nil {
handle.Close()
return nil, fmt.Errorf("failed to compile ARP BPF filter: %w", err)
}
if err := handle.SetBPF(filter); err != nil {
handle.Close()
return nil, fmt.Errorf("failed to set BPF filter: %w", err)
}
if err := handle.SetPromiscuous(true); err != nil {
handle.Close()
return nil, fmt.Errorf("failed to set promiscuous mode: %w", err)
}
return handle, nil
}

// sendARPRequests distributes IPs to worker goroutines for ARP request sending.
func (s *ARPScanner) sendARPRequests(ctx context.Context, handle *afpacket.TPacket, ips []net.IP) {
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

// arpWorker sends ARP requests for IPs from the channel.
func (s *ARPScanner) arpWorker(ctx context.Context, handle *afpacket.TPacket, ips <-chan net.IP, retries int) {
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
func (s *ARPScanner) sendARPRequest(handle *afpacket.TPacket, targetIP net.IP) error {
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
func (s *ARPScanner) listenForResponses(ctx context.Context, handle *afpacket.TPacket, results chan<- Device, seen *sync.Map) {
packetSource := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)
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
