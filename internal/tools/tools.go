// Package tools provides integrated network diagnostic tools.
// This package implements netcat, telnet, traceroute, dig, and whois functionality.
package tools

import (
	"context"
)

// NetworkTool defines the interface for network diagnostic tools
type NetworkTool interface {
	// Name returns the tool name
	Name() string

	// Execute runs the tool with given arguments
	Execute(ctx context.Context, args []string) (<-chan string, error)

	// Validate validates the arguments
	Validate(args []string) error
}
