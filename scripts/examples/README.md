# TuiMap Example Scripts

This directory contains example Tengo scripts demonstrating TuiMap's scripting capabilities.

**Note:** The scripting engine is planned for Phase 4 and is not yet implemented.

## Available Examples

### auto-scan.tengo
Automatic network scanning with alerting on:
- New device detection
- Suspicious open ports
- Device status changes

## Future Scripting Capabilities

Once implemented (Phase 4), scripts will be able to:

- Perform network scans with custom parameters
- Query and filter device information
- Create custom alerts and notifications
- Integrate with external systems
- Automate network monitoring tasks
- Store persistent data with key-value storage

## Script API Reference

See the PLAN.md Section 3.5 for the complete scripting API documentation.

### Network Functions
- `scan(subnet)` - Perform network scan
- `ping(ip)` - ICMP ping
- `portScan(ip, ports)` - TCP port scan
- `resolve(hostname)` - DNS lookup

### Device Management
- `getDevices()` - Get all devices
- `getDevice(ip)` - Get specific device
- `findDevices(filter)` - Filter devices

### Alert Functions
- `alert(message, severity)` - Create alert
- `alertDevice(ip, message, severity)` - Device-specific alert

### Storage Functions
- `set(key, value)` - Store data
- `get(key)` - Retrieve data
- `exists(key)` - Check if key exists

For complete API documentation, see [PLAN.md](../../PLAN.md).
