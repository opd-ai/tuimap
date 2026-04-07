package tools

import (
	"context"
	"net"
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
		nc.Validate(args)
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
		telnet.Validate(args)
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
		tr.Validate(args)
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
		dig.Validate(args)
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
		whois.Validate(args)
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
