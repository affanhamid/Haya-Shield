# Haya Shield

**macOS-only firewall-based adult content blocker.**  
Uses `pfctl`, Go, and `launchd` to dynamically block known domains by IP and persist across reboots.

---

## Getting Started

### 1. Build or use existing binary

The `tracker` binary is already built. It handles:

- Real-time packet sniffing
- Automatic IP blocking via PF
- Restoring config files on deletion

If you want to rebuild:

```bash
go build -o tracker cmd/tracker/main.go
```

---

### 2. Create the LaunchDaemon

Create the daemon file at:

```bash
sudo nano /Library/LaunchDaemons/com.hayashield.guardian.plist
```

Paste this (replace path):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.hayashield.guardian</string>

  <key>ProgramArguments</key>
  <array>
    <string>/full/path/to/tracker</string>
  </array>

  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>

  <key>StandardOutPath</key>
  <string>/var/log/haya-shield.log</string>
  <key>StandardErrorPath</key>
  <string>/var/log/haya-shield.err</string>
</dict>
</plist>
```

### 3. Load and start the service

```bash
sudo launchctl bootstrap system /Library/LaunchDaemons/com.hayashield.guardian.plist
```

---

### 4. View logs

Open two terminals:

```bash
sudo tail -f /var/log/haya-shield.log
sudo tail -f /var/log/haya-shield.err
```

You should see tracking and file restore logs.

---

## Project Layout

```
tracker                 # built binary
configs/
├─ pf.rules             # dynamic firewall rules
└─ blocked_ips.json     # blocked domain list (resolved to IPs)
```

---

## TODO

- [ ] **Make it unkillable**

  - `launchd` ensures auto-restart on crash
  - But `sudo launchctl bootout` still disables it
  - Solution: add a `watchdog` process that resurrects it

- [ ] **Prevent PF anchor tampering**

  - If user removes the `anchor` from `/etc/pf.conf`, blocking is bypassed
  - Possible solutions:
    - Add integrity checker for `/etc/pf.conf`
    - Back up & restore on change
    - Lock file via `chflags schg`

- [ ] **Block all protocols**
  - Currently blocks TCP, UDP, and ICMP
  - Needs QUIC (UDP 443) + IPv6 to be enforced everywhere

---

## Testing Reboot Persistence

```bash
sudo reboot
```

After reboot:

```bash
sudo launchctl print system/com.hayashield.guardian
```

It should show `state = running`.

---

## Known Limitations

| Limitation                     | Mitigation Plan                         |
| ------------------------------ | --------------------------------------- |
| `sudo launchctl bootout` works | Watchdog process that restarts guardian |
| User edits `/etc/pf.conf`      | Monitor and restore anchor section      |
| Deletion of binary or configs  | Already handled via file monitoring     |
| Blocking bypass via IPv6       | Support added, validation still needed  |

---

## License

Private use only. Developed as part of the Haya Project.

