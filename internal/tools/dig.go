// Package tools provides integrated network diagnostic tools.
package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// DigTool implements DNS lookup functionality.
type DigTool struct {
	timeout  time.Duration
	resolver *net.Resolver
}

// NewDigTool creates a new dig tool.
func NewDigTool(timeout time.Duration, dnsServer string) *DigTool {
	tool := &DigTool{timeout: timeout}

	if dnsServer != "" {
		tool.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", dnsServer)
			},
		}
	} else {
		tool.resolver = net.DefaultResolver
	}

	return tool
}

// Name returns the tool name.
func (t *DigTool) Name() string {
	return "dig"
}

// Validate validates the arguments.
func (t *DigTool) Validate(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dig <hostname> [type] [@server]")
	}
	return nil
}

// Execute runs the dig tool.
func (t *DigTool) Execute(ctx context.Context, args []string) (<-chan string, error) {
	if err := t.Validate(args); err != nil {
		return nil, err
	}

	hostname := args[0]
	queryType := "A"
	dnsServer := ""

	// Parse arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "@") {
			dnsServer = strings.TrimPrefix(arg, "@")
		} else {
			queryType = strings.ToUpper(arg)
		}
	}

	output := make(chan string, 100)

	go func() {
		defer close(output)

		output <- fmt.Sprintf("; <<>> dig %s %s\n", hostname, queryType)
		output <- ";; Query time: "

		start := time.Now()

		// Create context with timeout
		queryCtx, cancel := context.WithTimeout(ctx, t.timeout)
		defer cancel()

		// Create resolver if custom DNS server specified
		resolver := t.resolver
		if dnsServer != "" {
			if !strings.Contains(dnsServer, ":") {
				dnsServer = dnsServer + ":53"
			}
			resolver = &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: t.timeout}
					return d.DialContext(ctx, "udp", dnsServer)
				},
			}
			output <- fmt.Sprintf(";; SERVER: %s\n", dnsServer)
		}

		var results []string
		var err error

		switch queryType {
		case "A", "ANY":
			results, err = t.lookupA(queryCtx, resolver, hostname)
		case "AAAA":
			results, err = t.lookupAAAA(queryCtx, resolver, hostname)
		case "MX":
			results, err = t.lookupMX(queryCtx, resolver, hostname)
		case "TXT":
			results, err = t.lookupTXT(queryCtx, resolver, hostname)
		case "NS":
			results, err = t.lookupNS(queryCtx, resolver, hostname)
		case "CNAME":
			results, err = t.lookupCNAME(queryCtx, resolver, hostname)
		case "PTR":
			results, err = t.lookupPTR(queryCtx, resolver, hostname)
		default:
			err = fmt.Errorf("unsupported query type: %s", queryType)
		}

		queryTime := time.Since(start)
		output <- fmt.Sprintf("%v\n", queryTime.Round(time.Millisecond))

		if err != nil {
			output <- fmt.Sprintf(";; Error: %v\n", err)
			return
		}

		output <- "\n;; ANSWER SECTION:\n"
		for _, result := range results {
			output <- fmt.Sprintf("%s.\t\tIN\t%s\t%s\n", hostname, queryType, result)
		}

		output <- fmt.Sprintf("\n;; Query completed in %v\n", queryTime.Round(time.Millisecond))
	}()

	return output, nil
}

// lookupA performs an A record lookup.
func (t *DigTool) lookupA(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	ips, err := resolver.LookupIP(ctx, "ip4", hostname)
	if err != nil {
		return nil, err
	}

	results := make([]string, len(ips))
	for i, ip := range ips {
		results[i] = ip.String()
	}
	return results, nil
}

// lookupAAAA performs an AAAA record lookup.
func (t *DigTool) lookupAAAA(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	ips, err := resolver.LookupIP(ctx, "ip6", hostname)
	if err != nil {
		return nil, err
	}

	results := make([]string, len(ips))
	for i, ip := range ips {
		results[i] = ip.String()
	}
	return results, nil
}

// lookupMX performs an MX record lookup.
func (t *DigTool) lookupMX(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	mxs, err := resolver.LookupMX(ctx, hostname)
	if err != nil {
		return nil, err
	}

	results := make([]string, len(mxs))
	for i, mx := range mxs {
		results[i] = fmt.Sprintf("%d %s", mx.Pref, mx.Host)
	}
	return results, nil
}

// lookupTXT performs a TXT record lookup.
func (t *DigTool) lookupTXT(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	return resolver.LookupTXT(ctx, hostname)
}

// lookupNS performs an NS record lookup.
func (t *DigTool) lookupNS(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	nss, err := resolver.LookupNS(ctx, hostname)
	if err != nil {
		return nil, err
	}

	results := make([]string, len(nss))
	for i, ns := range nss {
		results[i] = ns.Host
	}
	return results, nil
}

// lookupCNAME performs a CNAME record lookup.
func (t *DigTool) lookupCNAME(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	cname, err := resolver.LookupCNAME(ctx, hostname)
	if err != nil {
		return nil, err
	}
	return []string{cname}, nil
}

// lookupPTR performs a reverse DNS lookup.
func (t *DigTool) lookupPTR(ctx context.Context, resolver *net.Resolver, hostname string) ([]string, error) {
	// For PTR, hostname is actually an IP address
	names, err := resolver.LookupAddr(ctx, hostname)
	if err != nil {
		return nil, err
	}
	return names, nil
}

// LookupIP performs a simple IP lookup.
func (t *DigTool) LookupIP(ctx context.Context, hostname string) ([]net.IP, error) {
	return t.resolver.LookupIP(ctx, "ip", hostname)
}
