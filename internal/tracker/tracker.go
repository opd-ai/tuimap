// Package tracker provides device state management and alert functionality.
// This package maintains real-time device state and triggers alerts based
// on configured rules.
package tracker

import (
	"time"

	"github.com/opd-ai/tuimap/internal/scanner"
)

// Alert represents a triggered alert
type Alert struct {
	Type      AlertType
	Device    scanner.Device
	Timestamp time.Time
	Message   string
	Severity  int
}

// AlertType represents different types of alerts
type AlertType string

const (
	AlertNewDevice     AlertType = "new_device"
	AlertDeviceOffline AlertType = "device_offline"
	AlertPortChange    AlertType = "port_change"
	AlertMACConflict   AlertType = "mac_conflict"
)

// AlertRule defines an alert condition and action
type AlertRule struct {
	Condition func(scanner.Device) bool
	Action    func(Alert)
}

// Tracker manages device state and alerts
type Tracker interface {
	// Update updates the device registry with new scan results
	Update(devices []scanner.Device) error

	// GetDevices returns all tracked devices
	GetDevices() []scanner.Device

	// GetDevice returns a specific device by IP
	GetDevice(ip string) (scanner.Device, error)

	// GetAlerts returns all alerts
	GetAlerts() []Alert
}

// TODO: Implement device tracker with in-memory registry
// TODO: Implement alert engine
// TODO: Implement persistence layer
