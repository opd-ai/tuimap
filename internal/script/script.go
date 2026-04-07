// Package script provides the embedded Tengo scripting engine.
// This package integrates d5/tengo for user-extensible automation.
package script

import (
	"context"
)

// Engine manages script execution
type Engine interface {
	// Run executes a script
	Run(ctx context.Context, script string) error

	// LoadFile loads and runs a script from file
	LoadFile(ctx context.Context, path string) error

	// Stop stops all running scripts
	Stop()
}
