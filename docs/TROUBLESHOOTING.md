# TuiMap Troubleshooting Guide

This guide helps diagnose and resolve common issues with TuiMap.

## Table of Contents

1. [Permission Issues](#permission-issues)
2. [Scanning Issues](#scanning-issues)
3. [Performance Problems](#performance-problems)
4. [Configuration Issues](#configuration-issues)
5. [TUI Display Issues](#tui-display-issues)
6. [Network Detection Issues](#network-detection-issues)
7. [Scripting Issues](#scripting-issues)
8. [Storage Issues](#storage-issues)
9. [Diagnostic Commands](#diagnostic-commands)

---

## Permission Issues

### "Permission denied" when scanning

**Symptom:** Error message about permission denied or raw socket access.

**Cause:** ARP and ICMP scanning require root/admin privileges for raw socket access.

**Solutions:**

1. **Run with sudo:**
   ```bash
   sudo tuimap
   ```

2. **Grant capabilities (Linux):**
   ```bash
   sudo setcap cap_net_raw+ep /usr/local/bin/tuimap
   ```

3. **Use TCP-only scanning (no root required):**
   Configure TCP-only scanning in `~/.config/tuimap/config.yaml`:
   ```yaml
   scanner:
     methods:
       - tcp
   ```

### Cannot open network interface

**Symptom:** Error about interface not found or access denied.

**Cause:** Interface name is incorrect or interface is not available.

**Solutions:**

1. **List available interfaces:**
   ```bash
   ip link show
   # or
   ifconfig -a
   ```

2. **Specify correct interface:**
   ```bash
   tuimap --interface eth0 scan
   ```

3. **Check interface is up:**
   ```bash
   ip link set eth0 up
   ```

---

## Scanning Issues

### No devices found

**Symptom:** Scan completes but no devices are discovered.

**Possible Causes:**
- Wrong subnet being scanned
- Firewall blocking packets
- Network interface issues
- All methods failing silently

**Solutions:**

1. **Verify subnet detection:**
   ```bash
   tuimap scan --debug
   # Look for detected subnet in output
   ```

2. **Specify subnet manually:**
   ```bash
   tuimap scan --subnet 192.168.1.0/24
   ```

3. **Check firewall rules:**
   ```bash
   # Linux
   sudo iptables -L -n | grep -i icmp
   sudo iptables -L -n | grep -i arp
   ```

4. **Test basic connectivity:**
   ```bash
   ping -c 3 192.168.1.1  # Try gateway
   arping -c 3 192.168.1.1  # ARP ping
   ```

5. **Try scanning with debug mode:**
   ```bash
   tuimap scan --debug
   ```
   Configure individual scan methods in `~/.config/tuimap/config.yaml`:
   ```yaml
   scanner:
     methods:
       - arp   # Try each method individually
   ```

### Scan takes too long

**Symptom:** Scans exceed the 10-second target.

**Possible Causes:**
- Large subnet size
- High network latency
- Too many workers causing congestion
- Slow DNS resolution

**Solutions:**

1. **Reduce subnet size:**
   ```bash
   tuimap scan --subnet 192.168.1.0/25  # Half the hosts
   ```

2. **Adjust worker counts:**
   ```yaml
   scanner:
     arp:
       workers: 128    # Reduce from 256
     icmp:
       workers: 128
     tcp:
       workers: 256    # Reduce from 512
   ```

3. **Increase timeouts (for high-latency networks):**
   ```yaml
   scanner:
     arp:
       timeout: 200ms
     icmp:
       timeout: 2s
     tcp:
       timeout: 1s
   ```

4. **Disable slow methods:**
   ```yaml
   scanner:
     methods:
       - arp      # Fastest
       - tcp      # Usually reliable
       # Remove icmp if it's slow
   ```

### Incomplete device detection

**Symptom:** Some known devices aren't detected.

**Possible Causes:**
- Devices on different subnet
- Firewall blocking probes
- Devices in sleep/power-save mode
- MAC filtering on network

**Solutions:**

1. **Enable all scan methods:**
   ```yaml
   scanner:
     methods:
       - arp
       - icmp
       - tcp
   ```

2. **Increase retry counts:**
   ```yaml
   scanner:
     arp:
       retries: 3
     icmp:
       count: 3
   ```

3. **Scan multiple subnets:**
   ```bash
   tuimap scan --all-subnets
   ```

4. **Add more TCP ports:**
   ```yaml
   scanner:
     tcp:
       ports: [22, 80, 443, 8080, 3389, 5900, 554, 1900]
   ```

---

## Performance Problems

### High CPU usage during scans

**Symptom:** CPU spikes to 100% during network scans.

**Solutions:**

1. **Reduce worker counts:**
   ```yaml
   scanner:
     arp:
       workers: 64
     icmp:
       workers: 64
     tcp:
       workers: 128
   ```

2. **Increase per-host timeouts (reduces concurrency):**
   ```yaml
   scanner:
     arp:
       timeout: 200ms
   ```

### High memory usage

**Symptom:** TuiMap uses excessive memory over time.

**Solutions:**

1. **Reduce history retention:**
   ```yaml
   storage:
     history_retention: 24h  # From 168h (7 days)
   ```

2. **Clear old data:**
   ```bash
   rm ~/.local/share/tuimap/tuimap.db
   tuimap config init
   ```

3. **Reduce device tracking scope:**
   ```yaml
   tracker:
     max_devices: 500  # Limit tracked devices
   ```

### TUI lag or stuttering

**Symptom:** UI is slow to respond or updates erratically.

**Solutions:**

1. **Reduce refresh rate:**
   ```yaml
   tui:
     refresh_rate: 15  # From 30 FPS
   ```

2. **Use simpler view:**
   - Press `2` for Device List (less rendering than Network Map)

3. **Reduce device count in display:**
   - Use filter (`f`) to show fewer devices

---

## Configuration Issues

### Configuration file not found

**Symptom:** Error about missing config file.

**Solutions:**

1. **Initialize configuration:**
   ```bash
   tuimap config init
   ```

2. **Check file location:**
   ```bash
   ls -la ~/.config/tuimap/config.yaml
   ```

3. **Create directory if missing:**
   ```bash
   mkdir -p ~/.config/tuimap
   tuimap config init
   ```

### Invalid configuration

**Symptom:** Error parsing configuration file.

**Solutions:**

1. **Validate YAML syntax:**
   ```bash
   yamllint ~/.config/tuimap/config.yaml
   # or
   python -c "import yaml; yaml.safe_load(open('$HOME/.config/tuimap/config.yaml'))"
   ```

2. **Reset to defaults:**
   ```bash
   rm ~/.config/tuimap/config.yaml
   tuimap config init
   ```

3. **View current config:**
   ```bash
   tuimap config show
   ```

### Settings not taking effect

**Symptom:** Configuration changes don't appear to work.

**Solutions:**

1. **Restart TuiMap** - Config is loaded at startup

2. **Check correct file:**
   ```bash
   tuimap config show  # Shows loaded config path
   ```

3. **Use command line override:**
   ```bash
   tuimap --interface eth0 --debug
   ```

---

## TUI Display Issues

### UI doesn't render correctly

**Symptom:** Characters misaligned, boxes broken, colors wrong.

**Solutions:**

1. **Set terminal to UTF-8:**
   ```bash
   export LANG=en_US.UTF-8
   export LC_ALL=en_US.UTF-8
   ```

2. **Use compatible terminal:**
   - Recommended: iTerm2, Alacritty, Kitty, GNOME Terminal
   - May have issues: older xterm, Windows CMD

3. **Set TERM variable:**
   ```bash
   export TERM=xterm-256color
   ```

4. **Resize terminal:**
   - Minimum: 80x24
   - Recommended: 120x40+

### Colors not displaying

**Symptom:** No colors or wrong colors in TUI.

**Solutions:**

1. **Check color support:**
   ```bash
   echo $TERM
   tput colors  # Should show 256
   ```

2. **Enable 256 colors:**
   ```bash
   export TERM=xterm-256color
   ```

3. **Disable colors in config:**
   ```yaml
   tui:
     theme: none  # Plain text mode
   ```

### Screen doesn't clear on exit

**Symptom:** TUI content remains after quitting.

**Solutions:**

1. **Reset terminal:**
   ```bash
   reset
   # or
   tput reset
   ```

2. **Use alternate screen buffer:**
   - This is usually automatic; ensure terminal supports it

---

## Network Detection Issues

### Wrong subnet detected

**Symptom:** TuiMap scans wrong network range.

**Solutions:**

1. **Specify subnet explicitly:**
   ```bash
   tuimap scan --subnet 192.168.1.0/24
   ```

2. **Specify interface:**
   ```bash
   tuimap scan --interface eth0
   ```

3. **Check routing table:**
   ```bash
   ip route
   # or
   netstat -rn
   ```

### Gateway not detected

**Symptom:** Gateway shows as unknown or wrong IP.

**Solutions:**

1. **Check system gateway:**
   ```bash
   ip route | grep default
   ```

2. **Verify network configuration:**
   ```bash
   ip addr show
   ```

### NAT detection fails

**Symptom:** NAT info unavailable or incorrect.

**Solutions:**

1. **Enable NAT detection:**
   ```yaml
   nat:
     detect: true
     upnp_enabled: true
     nat_pmp_enabled: true
   ```

2. **Check UPnP on router:**
   - Ensure UPnP is enabled in router settings

3. **Try different STUN servers:**
   ```yaml
   nat:
     stun_servers:
       - stun.l.google.com:19302
       - stun.cloudflare.com:3478
   ```

---

## Scripting Issues

### Script fails to run

**Symptom:** Error when executing script.

**Solutions:**

1. **Check script syntax:**
   ```bash
   # Tengo scripts should be valid Tengo syntax
   # Review your script for syntax errors before running
   ```

2. **Check API usage:**
   - Verify function names match documentation
   - Check argument types and counts

3. **Enable debug mode:**
   ```bash
   tuimap --debug
   ```

### Script timeout

**Symptom:** Script stops with timeout error.

**Solutions:**

1. **Increase timeout:**
   ```yaml
   scripting:
     max_execution_time: 60s  # From 30s
   ```

2. **Optimize script:**
   - Reduce loop iterations
   - Avoid unnecessary scans within scripts

### Storage functions not working

**Symptom:** `get()` returns undefined, `set()` doesn't persist.

**Solutions:**

1. **Check storage path:**
   ```yaml
   storage:
     database: ~/.local/share/tuimap/tuimap.db
   ```

2. **Verify database file:**
   ```bash
   ls -la ~/.local/share/tuimap/tuimap.db
   ```

3. **Reset storage:**
   ```bash
   rm ~/.local/share/tuimap/tuimap.db
   ```

---

## Storage Issues

### Database corruption

**Symptom:** Errors about database or storage failures.

**Solutions:**

1. **Backup and reset:**
   ```bash
   cp ~/.local/share/tuimap/tuimap.db ~/tuimap-backup.db
   rm ~/.local/share/tuimap/tuimap.db
   ```

2. **Check disk space:**
   ```bash
   df -h ~/.local/share/tuimap/
   ```

3. **Check file permissions:**
   ```bash
   ls -la ~/.local/share/tuimap/
   ```

### History not saving

**Symptom:** Device history is empty or resets.

**Solutions:**

1. **Check storage configuration:**
   ```yaml
   storage:
     database: ~/.local/share/tuimap/tuimap.db
     history_retention: 168h
   ```

2. **Verify write permissions:**
   ```bash
   touch ~/.local/share/tuimap/test
   rm ~/.local/share/tuimap/test
   ```

---

## Diagnostic Commands

### Debug mode

```bash
# Run with debug logging
tuimap --debug

# Debug specific command
tuimap scan --debug
```

### Version check

```bash
tuimap version
```

### Configuration validation

```bash
tuimap config show
```

### Network diagnostics

```bash
# Check interfaces
ip link show
ip addr show

# Check routing
ip route
netstat -rn

# Test connectivity
ping -c 3 gateway_ip
arping -c 3 gateway_ip
```

### Log review

```bash
# If logging is configured
tail -f ~/.local/share/tuimap/tuimap.log

# System logs (if running as service)
journalctl -u tuimap
```

### Test scan methods

```bash
# Run a scan with debug mode to see which methods are being used
tuimap scan --subnet 192.168.1.0/24 --debug
```

---

## Getting Help

If these solutions don't resolve your issue:

1. **Check GitHub Issues:**
   https://github.com/opd-ai/tuimap/issues

2. **Collect diagnostic information:**
   ```bash
   tuimap version
   tuimap config show
   ```

3. **Open new issue with:**
   - TuiMap version (`tuimap version`)
   - Operating system and version
   - Error messages (exact text)
   - Steps to reproduce
   - Configuration (without sensitive data)
   - Diagnostic report output
