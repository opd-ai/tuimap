// Package tracker provides device state management and alert functionality.
package tracker

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/opd-ai/tuimap/internal/scanner"
	bolt "go.etcd.io/bbolt"
)

var (
	devicesBucket = []byte("devices")
	alertsBucket  = []byte("alerts")
	historyBucket = []byte("history")
)

// Storage provides persistent storage for device history.
type Storage struct {
	db        *bolt.DB
	retention time.Duration
}

// deviceRecord is the serializable form of a device.
type deviceRecord struct {
	IP        string                 `json:"ip"`
	MAC       string                 `json:"mac"`
	Hostname  string                 `json:"hostname"`
	Vendor    string                 `json:"vendor"`
	Ports     []int                  `json:"ports"`
	LastSeen  time.Time              `json:"last_seen"`
	FirstSeen time.Time              `json:"first_seen"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// alertRecord is the serializable form of an alert.
type alertRecord struct {
	Type      string       `json:"type"`
	Device    deviceRecord `json:"device"`
	Timestamp time.Time    `json:"timestamp"`
	Message   string       `json:"message"`
	Severity  int          `json:"severity"`
}

// NewStorage creates a new storage instance with the given database path.
func NewStorage(dbPath string, retention time.Duration) (*Storage, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Open database
	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{devicesBucket, alertsBucket, historyBucket} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &Storage{
		db:        db,
		retention: retention,
	}, nil
}

// Close closes the database.
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveDevice saves a device to persistent storage.
func (s *Storage) SaveDevice(device scanner.Device) error {
	record := deviceToRecord(device)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal device: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(devicesBucket)
		return b.Put([]byte(device.IP.String()), data)
	})
}

// SaveDevices saves multiple devices to persistent storage.
func (s *Storage) SaveDevices(devices []scanner.Device) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(devicesBucket)
		for _, device := range devices {
			record := deviceToRecord(device)
			data, err := json.Marshal(record)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(device.IP.String()), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadDevices loads all devices from storage.
func (s *Storage) LoadDevices() ([]scanner.Device, error) {
	var devices []scanner.Device

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(devicesBucket)
		return b.ForEach(func(k, v []byte) error {
			var record deviceRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			device := recordToDevice(record)
			devices = append(devices, device)
			return nil
		})
	})

	return devices, err
}

// SaveAlert saves an alert to persistent storage.
func (s *Storage) SaveAlert(alert Alert) error {
	record := alertToRecord(alert)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	key := fmt.Sprintf("%d-%s", alert.Timestamp.UnixNano(), alert.Type)
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertsBucket)
		return b.Put([]byte(key), data)
	})
}

// LoadAlerts loads all alerts from storage.
func (s *Storage) LoadAlerts() ([]Alert, error) {
	var alerts []Alert

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertsBucket)
		return b.ForEach(func(k, v []byte) error {
			var record alertRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			alert := recordToAlert(record)
			alerts = append(alerts, alert)
			return nil
		})
	})

	return alerts, err
}

// Cleanup removes old records based on retention policy.
func (s *Storage) Cleanup() error {
	cutoff := time.Now().Add(-s.retention)

	return s.db.Update(func(tx *bolt.Tx) error {
		// Clean up old alerts
		ab := tx.Bucket(alertsBucket)
		c := ab.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var record alertRecord
			if err := json.Unmarshal(v, &record); err == nil {
				if record.Timestamp.Before(cutoff) {
					if err := ab.Delete(k); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// deviceToRecord converts a Device to a serializable record.
func deviceToRecord(d scanner.Device) deviceRecord {
	macStr := ""
	if d.MAC != nil {
		macStr = d.MAC.String()
	}
	return deviceRecord{
		IP:        d.IP.String(),
		MAC:       macStr,
		Hostname:  d.Hostname,
		Vendor:    d.Vendor,
		Ports:     d.Ports,
		LastSeen:  d.LastSeen,
		FirstSeen: d.FirstSeen,
		Status:    string(d.Status),
		Metadata:  d.Metadata,
	}
}

// recordToDevice converts a record back to a Device.
func recordToDevice(r deviceRecord) scanner.Device {
	var mac net.HardwareAddr
	if r.MAC != "" {
		mac, _ = net.ParseMAC(r.MAC)
	}
	return scanner.Device{
		IP:        net.ParseIP(r.IP),
		MAC:       mac,
		Hostname:  r.Hostname,
		Vendor:    r.Vendor,
		Ports:     r.Ports,
		LastSeen:  r.LastSeen,
		FirstSeen: r.FirstSeen,
		Status:    scanner.DeviceStatus(r.Status),
		Metadata:  r.Metadata,
	}
}

// alertToRecord converts an Alert to a serializable record.
func alertToRecord(a Alert) alertRecord {
	return alertRecord{
		Type:      string(a.Type),
		Device:    deviceToRecord(a.Device),
		Timestamp: a.Timestamp,
		Message:   a.Message,
		Severity:  a.Severity,
	}
}

// recordToAlert converts a record back to an Alert.
func recordToAlert(r alertRecord) Alert {
	return Alert{
		Type:      AlertType(r.Type),
		Device:    recordToDevice(r.Device),
		Timestamp: r.Timestamp,
		Message:   r.Message,
		Severity:  r.Severity,
	}
}
