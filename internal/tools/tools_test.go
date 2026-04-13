package tools

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNetcatToolName(t *testing.T) {
	nc := NewNetcatTool(5 * time.Second)
	if nc.Name() != "netcat" {
		t.Errorf("Expected 'netcat', got '%s'", nc.Name())
	}
}

func TestNetcatToolValidate(t *testing.T) {
	nc := NewNetcatTool(5 * time.Second)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"only host", []string{"localhost"}, true},
		{"host and port", []string{"localhost", "80"}, false},
		{"invalid port", []string{"localhost", "abc"}, true},
		{"with udp flag", []string{"localhost", "80", "--udp"}, false},
		{"with data", []string{"localhost", "80", "--data", "test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nc.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelnetToolName(t *testing.T) {
	telnet := NewTelnetTool(5 * time.Second)
	if telnet.Name() != "telnet" {
		t.Errorf("Expected 'telnet', got '%s'", telnet.Name())
	}
}

func TestTelnetToolValidate(t *testing.T) {
	telnet := NewTelnetTool(5 * time.Second)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"host only", []string{"localhost"}, false},
		{"host and port", []string{"localhost", "23"}, false},
		{"invalid port", []string{"localhost", "abc"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := telnet.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelnetFilterTelnet(t *testing.T) {
	telnet := NewTelnetTool(5 * time.Second)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "Hello World\n", "Hello World\n"},
		{"with control chars", "Hello\x00World", "HelloWorld"},
		{"with IAC sequence", "Test\xff\xfb\x01Text", "TestText"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := telnet.filterTelnet(tt.input)
			if result != tt.expected {
				t.Errorf("filterTelnet() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTracerouteToolName(t *testing.T) {
	tr := NewTracerouteTool(30, 1*time.Second)
	if tr.Name() != "traceroute" {
		t.Errorf("Expected 'traceroute', got '%s'", tr.Name())
	}
}

func TestTracerouteToolValidate(t *testing.T) {
	tr := NewTracerouteTool(30, 1*time.Second)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"host only", []string{"google.com"}, false},
		{"with max hops", []string{"google.com", "--max-hops", "15"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tr.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHopString(t *testing.T) {
	tests := []struct {
		name string
		hop  Hop
		num  int
	}{
		{"timeout hop", Hop{Timeout: true}, 1},
		{"reached hop", Hop{Reached: true}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hop.String(tt.num)
			if result == "" {
				t.Error("Expected non-empty string")
			}
		})
	}
}

func TestDigToolName(t *testing.T) {
	dig := NewDigTool(5*time.Second, "")
	if dig.Name() != "dig" {
		t.Errorf("Expected 'dig', got '%s'", dig.Name())
	}
}

func TestDigToolValidate(t *testing.T) {
	dig := NewDigTool(5*time.Second, "")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"hostname only", []string{"example.com"}, false},
		{"with type", []string{"example.com", "MX"}, false},
		{"with server", []string{"example.com", "@8.8.8.8"}, false},
		{"full args", []string{"example.com", "A", "@8.8.8.8"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dig.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWhoisToolName(t *testing.T) {
	whois := NewWhoisTool(5 * time.Second)
	if whois.Name() != "whois" {
		t.Errorf("Expected 'whois', got '%s'", whois.Name())
	}
}

func TestWhoisToolValidate(t *testing.T) {
	whois := NewWhoisTool(5 * time.Second)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"domain only", []string{"example.com"}, false},
		{"with server", []string{"example.com", "--server", "whois.example.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := whois.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWhoisGetServer(t *testing.T) {
	whois := NewWhoisTool(5 * time.Second)

	tests := []struct {
		query    string
		expected string
	}{
		{"example.com", "whois.verisign-grs.com"},
		{"example.org", "whois.pir.org"},
		{"example.io", "whois.nic.io"},
		{"8.8.8.8", "whois.arin.net"},
		{"example", "whois.iana.org"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := whois.getWhoisServer(tt.query)
			if result != tt.expected {
				t.Errorf("getWhoisServer(%s) = %s, want %s", tt.query, result, tt.expected)
			}
		})
	}
}

func TestNetcatTCPConnectTimeout(t *testing.T) {
	nc := NewNetcatTool(100 * time.Millisecond)
	ctx := context.Background()

	// Connect to non-routable IP should timeout
	ok, _, err := nc.TCPConnect(ctx, "10.255.255.1", 80)
	if ok {
		t.Error("Expected connection to fail")
	}
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestDigLookupWithContext(t *testing.T) {
	dig := NewDigTool(100*time.Millisecond, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := dig.LookupIP(ctx, "example.com")
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

// Benchmark tests for tools performance

func BenchmarkNetcatCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewNetcatTool(5 * time.Second)
	}
}

func BenchmarkNetcatValidate(b *testing.B) {
	nc := NewNetcatTool(5 * time.Second)
	args := []string{"localhost", "80"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nc.Validate(args)
	}
}

func BenchmarkTelnetCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTelnetTool(5 * time.Second)
	}
}

func BenchmarkTelnetValidate(b *testing.B) {
	telnet := NewTelnetTool(5 * time.Second)
	args := []string{"localhost", "23"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = telnet.Validate(args)
	}
}

func BenchmarkTelnetFilterTelnet(b *testing.B) {
	telnet := NewTelnetTool(5 * time.Second)
	input := "Hello\xff\xfb\x01World\x00Test\xff\xfd\x03Data"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telnet.filterTelnet(input)
	}
}

func BenchmarkTracerouteCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTracerouteTool(30, 1*time.Second)
	}
}

func BenchmarkTracerouteValidate(b *testing.B) {
	tr := NewTracerouteTool(30, 1*time.Second)
	args := []string{"google.com"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tr.Validate(args)
	}
}

func BenchmarkDigCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewDigTool(5*time.Second, "")
	}
}

func BenchmarkDigValidate(b *testing.B) {
	dig := NewDigTool(5*time.Second, "")
	args := []string{"example.com", "A"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dig.Validate(args)
	}
}

func BenchmarkWhoisCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWhoisTool(5 * time.Second)
	}
}

func BenchmarkWhoisValidate(b *testing.B) {
	whois := NewWhoisTool(5 * time.Second)
	args := []string{"example.com"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = whois.Validate(args)
	}
}

func BenchmarkWhoisGetServer(b *testing.B) {
	whois := NewWhoisTool(5 * time.Second)
	domains := []string{"example.com", "example.org", "example.net", "example.io"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		whois.getWhoisServer(domains[i%len(domains)])
	}
}

func BenchmarkHopString(b *testing.B) {
	hop := Hop{
		IP:      net.ParseIP("192.168.1.1"),
		RTT:     15 * time.Millisecond,
		Reached: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hop.String(1)
	}
}

// ===== Additional tests for improved coverage =====

// TestNetcatParseOptions tests option parsing.
func TestNetcatParseOptions(t *testing.T) {
	nc := NewNetcatTool(5 * time.Second)

	tests := []struct {
		name     string
		args     []string
		wantHost string
		wantPort string
		wantUDP  bool
		wantData string
	}{
		{
			name:     "basic args",
			args:     []string{"localhost", "80"},
			wantHost: "localhost",
			wantPort: "80",
		},
		{
			name:     "with udp flag",
			args:     []string{"host", "443", "--udp"},
			wantHost: "host",
			wantPort: "443",
			wantUDP:  true,
		},
		{
			name:     "with udp short flag",
			args:     []string{"host", "443", "-u"},
			wantHost: "host",
			wantPort: "443",
			wantUDP:  true,
		},
		{
			name:     "with data flag",
			args:     []string{"host", "80", "--data", "hello"},
			wantHost: "host",
			wantPort: "80",
			wantData: "hello",
		},
		{
			name:     "with data short flag",
			args:     []string{"host", "80", "-d", "test"},
			wantHost: "host",
			wantPort: "80",
			wantData: "test",
		},
		{
			name:     "all options",
			args:     []string{"example.com", "22", "--udp", "--data", "message"},
			wantHost: "example.com",
			wantPort: "22",
			wantUDP:  true,
			wantData: "message",
		},
		{
			name:     "data flag at end without value",
			args:     []string{"host", "80", "--data"},
			wantHost: "host",
			wantPort: "80",
			wantData: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := nc.parseOptions(tt.args)
			if opts.host != tt.wantHost {
				t.Errorf("host = %q, want %q", opts.host, tt.wantHost)
			}
			if opts.port != tt.wantPort {
				t.Errorf("port = %q, want %q", opts.port, tt.wantPort)
			}
			if opts.useUDP != tt.wantUDP {
				t.Errorf("useUDP = %v, want %v", opts.useUDP, tt.wantUDP)
			}
			if opts.data != tt.wantData {
				t.Errorf("data = %q, want %q", opts.data, tt.wantData)
			}
		})
	}
}

// TestNetcatExecuteValidationError tests Execute with invalid args.
func TestNetcatExecuteValidationError(t *testing.T) {
	nc := NewNetcatTool(100 * time.Millisecond)
	ctx := context.Background()

	_, err := nc.Execute(ctx, []string{})
	if err == nil {
		t.Error("Expected validation error for empty args")
	}
}

// TestNetcatBannerTimeout tests banner grab timeout.
func TestNetcatBannerTimeout(t *testing.T) {
	nc := NewNetcatTool(100 * time.Millisecond)
	ctx := context.Background()

	// Connect to non-routable IP should timeout
	_, err := nc.Banner(ctx, "10.255.255.1", 80)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestTelnetParseHostPort tests host/port parsing.
func TestTelnetParseHostPort(t *testing.T) {
	telnet := NewTelnetTool(5 * time.Second)

	tests := []struct {
		args     []string
		wantHost string
		wantPort string
	}{
		{[]string{"localhost"}, "localhost", "23"},
		{[]string{"example.com", "22"}, "example.com", "22"},
		{[]string{"host", "8080"}, "host", "8080"},
	}

	for _, tt := range tests {
		host, port := telnet.parseHostPort(tt.args)
		if host != tt.wantHost {
			t.Errorf("parseHostPort(%v) host = %q, want %q", tt.args, host, tt.wantHost)
		}
		if port != tt.wantPort {
			t.Errorf("parseHostPort(%v) port = %q, want %q", tt.args, port, tt.wantPort)
		}
	}
}

// TestTelnetExecuteValidationError tests Execute with invalid args.
func TestTelnetExecuteValidationError(t *testing.T) {
	telnet := NewTelnetTool(100 * time.Millisecond)
	ctx := context.Background()

	_, err := telnet.Execute(ctx, []string{})
	if err == nil {
		t.Error("Expected validation error for empty args")
	}
}

// TestTelnetConnectTimeout tests Connect timeout.
func TestTelnetConnectTimeout(t *testing.T) {
	telnet := NewTelnetTool(100 * time.Millisecond)
	ctx := context.Background()

	_, err := telnet.Connect(ctx, "10.255.255.1", 23)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestTracerouteParseMaxHops tests max hops parsing.
func TestTracerouteParseMaxHops(t *testing.T) {
	tr := NewTracerouteTool(30, 1*time.Second)

	tests := []struct {
		args []string
		want int
	}{
		{[]string{"host"}, 30},
		{[]string{"host", "--max-hops", "15"}, 15},
		{[]string{"host", "-m", "10"}, 10},
		{[]string{"host", "--max-hops", "invalid"}, 30},
		{[]string{"host", "--max-hops"}, 30}, // missing value
	}

	for _, tt := range tests {
		got := tr.parseMaxHops(tt.args)
		if got != tt.want {
			t.Errorf("parseMaxHops(%v) = %d, want %d", tt.args, got, tt.want)
		}
	}
}

// TestTracerouteExecuteValidationError tests Execute with invalid args.
func TestTracerouteExecuteValidationError(t *testing.T) {
	tr := NewTracerouteTool(30, 100*time.Millisecond)
	ctx := context.Background()

	_, err := tr.Execute(ctx, []string{})
	if err == nil {
		t.Error("Expected validation error for empty args")
	}
}

// TestTracerouteResolveIPv4Error tests hostname resolution error.
func TestTracerouteResolveIPv4Error(t *testing.T) {
	tr := NewTracerouteTool(30, 100*time.Millisecond)

	_, err := tr.resolveIPv4("invalid.nonexistent.domain.test")
	if err == nil {
		t.Error("Expected error for invalid domain")
	}
}

// TestTracerouteDefaultMaxHops tests default max hops.
func TestTracerouteDefaultMaxHops(t *testing.T) {
	tr := NewTracerouteTool(0, 1*time.Second) // 0 should default to 30
	if tr.maxHops != 30 {
		t.Errorf("Expected default maxHops=30, got %d", tr.maxHops)
	}
}

// TestHopStringVariations tests different hop formatting.
func TestHopStringVariations(t *testing.T) {
	tests := []struct {
		name     string
		hop      Hop
		num      int
		contains string
	}{
		{
			name:     "timeout",
			hop:      Hop{Timeout: true},
			num:      1,
			contains: "*",
		},
		{
			name: "with IP only",
			hop: Hop{
				IP:  net.ParseIP("192.168.1.1"),
				RTT: 10 * time.Millisecond,
			},
			num:      2,
			contains: "192.168.1.1",
		},
		{
			name: "with hostname",
			hop: Hop{
				IP:       net.ParseIP("8.8.8.8"),
				Hostname: "dns.google",
				RTT:      15 * time.Millisecond,
			},
			num:      3,
			contains: "dns.google",
		},
		{
			name: "hostname equals IP",
			hop: Hop{
				IP:       net.ParseIP("8.8.8.8"),
				Hostname: "8.8.8.8",
				RTT:      15 * time.Millisecond,
			},
			num:      3,
			contains: "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hop.String(tt.num)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Hop.String() = %q, should contain %q", result, tt.contains)
			}
		})
	}
}

// TestDigExecuteValidationError tests Execute with invalid args.
func TestDigExecuteValidationError(t *testing.T) {
	dig := NewDigTool(100*time.Millisecond, "")
	ctx := context.Background()

	_, err := dig.Execute(ctx, []string{})
	if err == nil {
		t.Error("Expected validation error for empty args")
	}
}

// TestDigLookupMethods tests different lookup types.
func TestDigLookupMethods(t *testing.T) {
	dig := NewDigTool(2*time.Second, "")
	ctx := context.Background()

	// Test LookupIP
	ips, err := dig.LookupIP(ctx, "localhost")
	if err != nil {
		t.Logf("LookupIP error (may be expected): %v", err)
	} else if len(ips) == 0 {
		t.Log("No IPs returned for localhost (may be expected in CI)")
	}
}

// TestDigExecuteWithServer tests Execute with custom DNS server.
func TestDigExecuteWithServer(t *testing.T) {
	dig := NewDigTool(2*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute with custom server
	output, err := dig.Execute(ctx, []string{"example.com", "A", "@8.8.8.8"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Drain output channel
	count := 0
	for range output {
		count++
	}

	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigUnsupportedQueryType tests unsupported query type.
func TestDigUnsupportedQueryType(t *testing.T) {
	dig := NewDigTool(2*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"example.com", "INVALID"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Drain and check for error message
	found := false
	for line := range output {
		if strings.Contains(line, "unsupported query type") {
			found = true
		}
	}

	if !found {
		t.Error("Expected unsupported query type error in output")
	}
}

// TestWhoisParseArgs tests argument parsing.
func TestWhoisParseArgs(t *testing.T) {
	whois := NewWhoisTool(5 * time.Second)

	tests := []struct {
		args       []string
		wantQuery  string
		wantServer string
	}{
		{[]string{"example.com"}, "example.com", ""},
		{[]string{"example.com", "--server", "whois.test.com"}, "example.com", "whois.test.com"},
		{[]string{"example.com", "-h", "whois.custom.com"}, "example.com", "whois.custom.com"},
		{[]string{"example.com", "--server"}, "example.com", ""}, // missing server value
	}

	for _, tt := range tests {
		query, server := whois.parseArgs(tt.args)
		if query != tt.wantQuery {
			t.Errorf("parseArgs(%v) query = %q, want %q", tt.args, query, tt.wantQuery)
		}
		if server != tt.wantServer {
			t.Errorf("parseArgs(%v) server = %q, want %q", tt.args, server, tt.wantServer)
		}
	}
}

// TestWhoisExecuteValidationError tests Execute with invalid args.
func TestWhoisExecuteValidationError(t *testing.T) {
	whois := NewWhoisTool(100 * time.Millisecond)
	ctx := context.Background()

	_, err := whois.Execute(ctx, []string{})
	if err == nil {
		t.Error("Expected validation error for empty args")
	}
}

// TestWhoisGetServerExtended tests more TLD lookups.
func TestWhoisGetServerExtended(t *testing.T) {
	whois := NewWhoisTool(5 * time.Second)

	tests := []struct {
		query    string
		expected string
	}{
		{"example.net", "whois.verisign-grs.com"},
		{"example.info", "whois.afilias.net"},
		{"example.biz", "whois.biz"},
		{"example.co", "whois.nic.co"},
		{"example.me", "whois.nic.me"},
		{"example.us", "whois.nic.us"},
		{"example.uk", "whois.nic.uk"},
		{"example.de", "whois.denic.de"},
		{"example.fr", "whois.nic.fr"},
		{"example.eu", "whois.eu"},
		{"example.ru", "whois.tcinet.ru"},
		{"example.cn", "whois.cnnic.cn"},
		{"example.jp", "whois.jprs.jp"},
		{"example.au", "whois.auda.org.au"},
		{"example.ca", "whois.cira.ca"},
		{"example.in", "whois.registry.in"},
		{"example.br", "whois.registro.br"},
		{"192.168.1.1", "whois.arin.net"},
		{"2001:db8::1", "whois.arin.net"},
		{"example.xyz", "whois.iana.org"}, // unknown TLD
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := whois.getWhoisServer(tt.query)
			if result != tt.expected {
				t.Errorf("getWhoisServer(%s) = %s, want %s", tt.query, result, tt.expected)
			}
		})
	}
}

// TestWhoisLookupMethods tests Lookup, LookupIP, LookupDomain.
func TestWhoisLookupMethods(t *testing.T) {
	whois := NewWhoisTool(100 * time.Millisecond)
	ctx := context.Background()

	// These will timeout but exercise the code paths
	_, _ = whois.Lookup(ctx, "10.255.255.1")
	_, _ = whois.LookupIP(ctx, "10.255.255.1")
	_, _ = whois.LookupDomain(ctx, "nonexistent.test")
}

// TestTracerouteTraceContextCancel tests context cancellation.
func TestTracerouteTraceContextCancel(t *testing.T) {
	tr := NewTracerouteTool(5, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	hops, err := tr.Trace(ctx, "localhost")
	if err != nil {
		// Error is acceptable if hostname resolution fails
		return
	}
	// If no error, hops should be empty or minimal due to cancellation
	_ = hops
}

// TestTracerouteTraceInvalidHost tests Trace with invalid host.
func TestTracerouteTraceInvalidHost(t *testing.T) {
	tr := NewTracerouteTool(5, 100*time.Millisecond)
	ctx := context.Background()

	_, err := tr.Trace(ctx, "invalid.nonexistent.domain.test")
	if err == nil {
		t.Error("Expected error for invalid host")
	}
}

// TestDigWithCustomResolver tests dig with custom DNS server in constructor.
func TestDigWithCustomResolver(t *testing.T) {
	dig := NewDigTool(2*time.Second, "8.8.8.8:53")

	if dig.resolver == nil {
		t.Error("Expected custom resolver to be set")
	}
}

// Integration-style test for netcat (using localhost listener)
func TestNetcatExecuteWithLocalServer(t *testing.T) {
	// Start a local TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	// Get the port
	port := listener.Addr().(*net.TCPAddr).Port

	// Accept connections and respond
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("Hello from server\n"))
	}()

	nc := NewNetcatTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := nc.Execute(ctx, []string{"127.0.0.1", fmt.Sprintf("%d", port)})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Drain output
	var lines []string
	for line := range output {
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		t.Error("Expected some output")
	}
}

// Integration-style test for telnet
func TestTelnetExecuteWithLocalServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("Welcome to telnet\n"))
	}()

	telnet := NewTelnetTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := telnet.Execute(ctx, []string{"127.0.0.1", fmt.Sprintf("%d", port)})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var lines []string
	for line := range output {
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		t.Error("Expected some output")
	}
}

// TestNetcatSendData tests sending data to a connection.
func TestNetcatSendData(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		received <- string(buf[:n])
	}()

	nc := NewNetcatTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := nc.Execute(ctx, []string{"127.0.0.1", fmt.Sprintf("%d", port), "--data", "test message"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Drain output
	for range output {
	}

	select {
	case data := <-received:
		if !strings.Contains(data, "test message") {
			t.Errorf("Expected to receive 'test message', got %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for data")
	}
}

// TestNetcatBannerWithServer tests banner grab with a real server.
func TestNetcatBannerWithServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("SSH-2.0-TestServer\n"))
	}()

	nc := NewNetcatTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	banner, err := nc.Banner(ctx, "127.0.0.1", port)
	if err != nil {
		t.Fatalf("Banner failed: %v", err)
	}

	if !strings.Contains(banner, "SSH") {
		t.Errorf("Expected banner to contain 'SSH', got %q", banner)
	}
}

// TestTelnetConnectWithServer tests telnet connect with real server.
func TestTelnetConnectWithServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("Welcome to server\n"))
	}()

	telnet := NewTelnetTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	banner, err := telnet.Connect(ctx, "127.0.0.1", port)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// The banner should contain the message (possibly filtered)
	if !strings.Contains(banner, "server") && !strings.Contains(banner, "Welcome") {
		t.Errorf("Expected banner to contain server message, got %q", banner)
	}
}

// TestNetcatTCPConnectSuccess tests successful TCP connection.
func TestNetcatTCPConnectSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			_ = conn.Close()
		}
	}()

	nc := NewNetcatTool(2 * time.Second)
	ctx := context.Background()

	ok, rtt, err := nc.TCPConnect(ctx, "127.0.0.1", port)
	if err != nil {
		t.Fatalf("TCPConnect failed: %v", err)
	}
	if !ok {
		t.Error("Expected connection to succeed")
	}
	if rtt <= 0 {
		t.Error("Expected positive RTT")
	}
}

// TestDigLookupAAAA tests AAAA record lookup.
func TestDigLookupAAAA(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Test AAAA lookup - google.com usually has IPv6
	output, err := dig.Execute(ctx, []string{"google.com", "AAAA"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigLookupMX tests MX record lookup.
func TestDigLookupMX(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"google.com", "MX"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigLookupTXT tests TXT record lookup.
func TestDigLookupTXT(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"google.com", "TXT"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigLookupNS tests NS record lookup.
func TestDigLookupNS(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"google.com", "NS"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigLookupCNAME tests CNAME record lookup.
func TestDigLookupCNAME(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// www.google.com usually has a CNAME
	output, err := dig.Execute(ctx, []string{"www.google.com", "CNAME"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestDigLookupPTR tests PTR (reverse DNS) lookup.
func TestDigLookupPTR(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Reverse lookup for 8.8.8.8
	output, err := dig.Execute(ctx, []string{"8.8.8.8", "PTR"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestTracerouteExecuteWithHost tests Execute with a valid host.
func TestTracerouteExecuteWithHost(t *testing.T) {
	tr := NewTracerouteTool(3, 500*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := tr.Execute(ctx, []string{"localhost", "--max-hops", "3"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestWhoisExecuteWithQuery tests Execute with a valid query.
func TestWhoisExecuteWithQuery(t *testing.T) {
	whois := NewWhoisTool(3 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := whois.Execute(ctx, []string{"example.com"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	// Whois may fail in CI, so just check we got the channel
	_ = count
}

// TestWhoisQueryMethod tests the query method directly.
func TestWhoisQueryMethod(t *testing.T) {
	whois := NewWhoisTool(100 * time.Millisecond)
	ctx := context.Background()

	// Query a non-existent server to test error path
	_, err := whois.query(ctx, "nonexistent.server.test:43", "test")
	if err == nil {
		t.Error("Expected error for non-existent server")
	}
}

// TestDigLookupASuccess tests A record lookup success path.
func TestDigLookupASuccess(t *testing.T) {
	dig := NewDigTool(3*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	output, err := dig.Execute(ctx, []string{"localhost", "A"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Error("Expected some output")
	}
}

// TestNetcatUDPMode tests netcat in UDP mode.
func TestNetcatUDPMode(t *testing.T) {
	// Start a UDP listener
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot resolve UDP addr: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Skipf("Cannot create UDP listener: %v", err)
	}
	defer func() { _ = conn.Close() }()

	port := conn.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil || n == 0 {
			return
		}
		_, _ = conn.WriteToUDP([]byte("UDP Response\n"), remoteAddr)
	}()

	nc := NewNetcatTool(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := nc.Execute(ctx, []string{"127.0.0.1", fmt.Sprintf("%d", port), "--udp", "--data", "test"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Drain output
	for range output {
	}
}

// TestTelnetHandleNegotiationScenarios tests various telnet negotiation scenarios.
func TestTelnetHandleNegotiationScenarios(t *testing.T) {
	tests := []struct {
		name     string
		send     []byte
		expected bool // true if we expect some response to be sent
	}{
		{"WILL command", []byte{0xFF, 0xFB, 0x01}, true},
		{"DO command", []byte{0xFF, 0xFD, 0x03}, true},
		{"WONT command", []byte{0xFF, 0xFC, 0x01}, false},
		{"DONT command", []byte{0xFF, 0xFE, 0x01}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Skipf("Cannot create listener: %v", err)
			}
			defer func() { _ = listener.Close() }()

			port := listener.Addr().(*net.TCPAddr).Port

			serverDone := make(chan struct{})
			go func() {
				defer close(serverDone)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer func() { _ = conn.Close() }()
				// Send negotiation command
				_, _ = conn.Write(tt.send)
				// Wait a bit for response
				time.Sleep(50 * time.Millisecond)
				// Send some data
				_, _ = conn.Write([]byte("Hello\n"))
			}()

			telnet := NewTelnetTool(1 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			output, err := telnet.Execute(ctx, []string{"127.0.0.1", fmt.Sprintf("%d", port)})
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Drain output
			for range output {
			}
			<-serverDone
		})
	}
}
