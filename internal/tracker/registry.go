// Package tracker provides device state management and alert functionality.
package tracker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/opd-ai/tuimap/internal/scanner"
)

// Registry maintains the in-memory device state with thread-safe access.
type Registry struct {
	devices   map[string]*scanner.Device
	mu        sync.RWMutex
	alertChan chan Alert
	rules     []AlertRule
	offline   time.Duration
}

// NewRegistry creates a new device registry.
func NewRegistry(offlineThreshold time.Duration) *Registry {
	if offlineThreshold == 0 {
		offlineThreshold = 5 * time.Minute
	}
	return &Registry{
		devices:   make(map[string]*scanner.Device),
		alertChan: make(chan Alert, 100),
		rules:     make([]AlertRule, 0),
		offline:   offlineThreshold,
	}
}

// Update updates the device registry with new scan results.
func (r *Registry) Update(devices []scanner.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	seenIPs := make(map[string]bool)

	for i := range devices {
		device := &devices[i]
		key := device.IP.String()
		seenIPs[key] = true

		existing, exists := r.devices[key]
		if !exists {
			// New device detected
			device.Status = scanner.StatusNew
			device.FirstSeen = now
			device.LastSeen = now
			r.devices[key] = device
			r.triggerAlert(AlertNewDevice, *device, "New device detected")
		} else {
			// Existing device - check for changes
			changed := r.detectChanges(existing, device)
			device.FirstSeen = existing.FirstSeen
			device.LastSeen = now

			if existing.Status == scanner.StatusOffline {
				device.Status = scanner.StatusOnline
				r.triggerAlert(AlertNewDevice, *device, "Device came back online")
			} else if changed {
				device.Status = scanner.StatusChanged
				r.triggerAlert(AlertPortChange, *device, "Device configuration changed")
			} else {
				device.Status = scanner.StatusOnline
			}
			r.devices[key] = device
		}
	}

	// Mark devices not seen as offline
	for key, device := range r.devices {
		if !seenIPs[key] && device.Status != scanner.StatusOffline {
			if time.Since(device.LastSeen) > r.offline {
				device.Status = scanner.StatusOffline
				r.triggerAlert(AlertDeviceOffline, *device, "Device went offline")
			}
		}
	}

	return nil
}

// detectChanges checks if device configuration has changed.
func (r *Registry) detectChanges(old, new *scanner.Device) bool {
	// Check MAC change (potential spoofing)
	if old.MAC != nil && new.MAC != nil {
		if old.MAC.String() != new.MAC.String() {
			r.triggerAlert(AlertMACConflict, *new, "MAC address changed")
			return true
		}
	}

	// Check port changes
	if len(old.Ports) != len(new.Ports) {
		return true
	}
	oldPorts := make(map[int]bool)
	for _, p := range old.Ports {
		oldPorts[p] = true
	}
	for _, p := range new.Ports {
		if !oldPorts[p] {
			return true
		}
	}

	return false
}

// triggerAlert sends an alert through the channel.
func (r *Registry) triggerAlert(alertType AlertType, device scanner.Device, msg string) {
	alert := Alert{
		Type:      alertType,
		Device:    device,
		Timestamp: time.Now(),
		Message:   msg,
		Severity:  getSeverity(alertType),
	}

	// Non-blocking send
	select {
	case r.alertChan <- alert:
	default:
		// Channel full, drop alert
	}
}

// getSeverity returns the default severity for an alert type.
func getSeverity(alertType AlertType) int {
	switch alertType {
	case AlertMACConflict:
		return 3 // High
	case AlertNewDevice:
		return 2 // Medium
	case AlertDeviceOffline:
		return 1 // Low
	case AlertPortChange:
		return 2 // Medium
	default:
		return 1
	}
}

// GetDevices returns all tracked devices.
func (r *Registry) GetDevices() []scanner.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]scanner.Device, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, *device)
	}
	return devices
}

// GetDevice returns a specific device by IP.
func (r *Registry) GetDevice(ip string) (scanner.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, exists := r.devices[ip]
	if !exists {
		return scanner.Device{}, fmt.Errorf("device not found: %s", ip)
	}
	return *device, nil
}

// GetAlerts returns all pending alerts and clears them.
func (r *Registry) GetAlerts() []Alert {
	alerts := make([]Alert, 0)
	for {
		select {
		case alert := <-r.alertChan:
			alerts = append(alerts, alert)
		default:
			return alerts
		}
	}
}

// AlertChan returns the alert channel for real-time monitoring.
func (r *Registry) AlertChan() <-chan Alert {
	return r.alertChan
}

// AddRule adds an alert rule.
func (r *Registry) AddRule(rule AlertRule) {
	r.rules = append(r.rules, rule)
}

// Count returns the number of tracked devices.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// OnlineCount returns the number of online devices.
func (r *Registry) OnlineCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, device := range r.devices {
		if device.Status == scanner.StatusOnline || device.Status == scanner.StatusNew {
			count++
		}
	}
	return count
}

// Export exports all devices as JSON.
func (r *Registry) Export() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]scanner.Device, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, *device)
	}

	return json.MarshalIndent(devices, "", "  ")
}

// Clear removes all devices from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices = make(map[string]*scanner.Device)
}
