# TuiMap Example Scripts

This directory contains example Tengo scripts demonstrating TuiMap's scripting capabilities.

## Available Examples

### auto-scan.tengo
Automatic network scanning with alerting on:
- New device detection
- Suspicious open ports
- Device status changes

### port-monitor.tengo
Port monitoring script that:
- Defines expected ports per device
- Alerts on unexpected open ports
- Alerts on missing expected ports
- Monitors critical ports on unknown devices

### device-inventory.tengo
Device inventory management that:
- Lists all tracked devices
- Counts devices by status
- Tracks changes over time
- Stores counts in persistent storage

### new-device-watcher.tengo
New device detection script that:
- Identifies newly discovered devices
- Logs detailed device information
- Generates alerts for new devices
- Tracks total new devices over time

### health-check.tengo
Network health diagnostics that:
- Runs a network scan
- Calculates online/offline ratios
- Checks hostname resolution rates
- Verifies scan time targets
- Generates overall health score

## Script API Reference

### Network Functions
- `scan()` - Perform network scan, returns result object
- `ping(ip)` - ICMP ping, returns latency
- `port_scan(ip, ports)` - TCP port scan
- `resolve(hostname)` - DNS lookup

### Device Management
- `get_devices()` - Get all tracked devices
- `get_device(ip)` - Get specific device by IP

### Alert Functions
- `alert(type, message)` - Create alert with type and message

### Storage Functions
- `set(key, value)` - Store persistent data
- `get(key)` - Retrieve data (returns undefined if not found)
- `delete(key)` - Remove data

### Device Object Properties
```tengo
device.ip         // IP address (string)
device.mac        // MAC address (string, may be empty)
device.hostname   // Hostname (string, may be empty)
device.vendor     // Vendor name (string, may be empty)
device.ports      // Open ports (array of integers)
device.status     // Status: "online", "offline", "new", "changed"
device.first_seen // First discovery timestamp
device.last_seen  // Last seen timestamp
```

### Scan Result Object
```tengo
result.devices    // Array of discovered devices
result.scan_time  // Scan duration in milliseconds
result.method     // Scan method used
```

## Running Scripts

```bash
# Run a script
tuimap script run scripts/examples/auto-scan.tengo

# Run with TUI script console
tuimap  # Then press '4' for Script Console
```

## Writing Your Own Scripts

1. Create a `.tengo` file in `~/.config/tuimap/scripts/`
2. Use the API functions documented above
3. Test with `tuimap script run your-script.tengo`

### Best Practices

- Use descriptive comments
- Handle undefined values (e.g., `if hostname != undefined`)
- Store state for tracking changes over time
- Use appropriate alert types: "new_device", "device_offline", "port_change", "mac_conflict"
- Keep scripts focused on single tasks

### Script Limits

| Limit | Value | Purpose |
|-------|-------|---------|
| Execution time | 30s | Prevent runaway scripts |
| Memory | 50MB | Prevent memory exhaustion |
| File access | None | Security sandboxing |
| System commands | None | Security isolation |

For complete API documentation, see [API.md](../../docs/API.md).
