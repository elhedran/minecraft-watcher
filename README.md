# Minecraft Server Auto-Shutdown Monitor

A lightweight Linux daemon that monitors a Minecraft server and automatically shuts down the entire system when idle to save cloud costs.

## Features

- Monitors Minecraft server using the [Server Management Protocol](https://minecraft.wiki/w/Minecraft_Server_Management_Protocol)
- Automatic **system shutdown** when:
  - Server has been running for more than 30 minutes
  - No players have been online for the last 10 minutes
- Runs as a systemd service
- Low resource footprint
- Single binary with no runtime dependencies
- Ideal for cloud environments (AWS EC2, etc.) to minimize costs

## Requirements

- Linux system with systemd
- Go 1.13 or later (for building)
- Minecraft server with Management Protocol enabled
- Access to the server's management API endpoint and secret

## Minecraft Server Configuration

Enable the Management Protocol in your server's `server.properties`:

```properties
management-server-enabled=true
management-server-host=localhost
management-server-port=25566
management-server-secret=<40-character-alphanumeric-secret>
```

The secret will be auto-generated if empty on first startup.

## Building

### Local Build (same architecture)

```bash
# Clone the repository
git clone https://github.com/ianw/minecraft-watcher.git
cd minecraft-watcher

# Build the binary
go build -o minecraft-watcher ./cmd/minecraft-watcher

# Or build with optimizations for production
go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher
```

### Cross-Compile for ARM-based AWS EC2 (Graviton)

If you're building on a different architecture (e.g., x86_64) for deployment on ARM-based EC2 instances (e.g., t4g, c7g, m7g):

```bash
# Build for ARM64 (AWS Graviton processors)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher

# Or for 32-bit ARM (older ARM instances)
GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher
```

### Architecture-Specific Builds

```bash
# For x86_64 / AMD64 (standard Intel/AMD servers)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher

# For ARM64 (AWS Graviton, Raspberry Pi 64-bit, etc.)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher

# For 32-bit ARM (Raspberry Pi 32-bit, older ARM devices)
GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher
```

**Verify target architecture on EC2:**
```bash
uname -m
# Output:
#   x86_64  -> use GOARCH=amd64
#   aarch64 -> use GOARCH=arm64
#   armv7l  -> use GOARCH=arm
```

The resulting binary is self-contained and can be deployed to any Linux system matching the target architecture.

## Installation

### 1. Build and Copy Binary

```bash
# Build the binary
go build -ldflags="-s -w" -o minecraft-watcher ./cmd/minecraft-watcher

# Copy to system location
sudo cp minecraft-watcher /usr/local/bin/
sudo chmod +x /usr/local/bin/minecraft-watcher
```

### 2. Create Configuration File

```bash
# Create config directory
sudo mkdir -p /etc/minecraft-watcher

# Create configuration file
sudo tee /etc/minecraft-watcher/config << EOF
MINECRAFT_MGMT_HOST=localhost
MINECRAFT_MGMT_PORT=25566
MINECRAFT_MGMT_SECRET=your-40-character-secret-here
MINECRAFT_MGMT_TLS_ENABLED=true
EOF

sudo chmod 600 /etc/minecraft-watcher/config
```

### 3. Create User (if needed)

```bash
# Create a dedicated user for the service
sudo useradd -r -s /bin/false minecraft
```

### 4. Configure Shutdown Permissions

**IMPORTANT:** The watcher needs permission to shut down the system when idle conditions are met.

```bash
# Install sudoers configuration
sudo cp deployments/minecraft-watcher-sudoers /etc/sudoers.d/minecraft-watcher
sudo chmod 440 /etc/sudoers.d/minecraft-watcher

# Verify syntax (important!)
sudo visudo -c
```

This grants the `minecraft` user passwordless sudo access to execute `systemctl poweroff` only.

**Security Note:** The watcher will shut down the entire system (not just the Minecraft server) when idle conditions are met. Always test with `TEST_MODE=true` first to verify the shutdown logic before deploying to production.

### 5. Install Systemd Service

```bash
# Copy service file
sudo cp deployments/minecraft-watcher.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable service to start at boot
sudo systemctl enable minecraft-watcher

# Start the service
sudo systemctl start minecraft-watcher

# Check status
sudo systemctl status minecraft-watcher
```

### 6. View Logs

```bash
# Follow logs in real-time
sudo journalctl -u minecraft-watcher -f

# View recent logs
sudo journalctl -u minecraft-watcher -n 50
```

## Configuration

Configuration is done via environment variables, either in `/etc/minecraft-watcher/config` or directly in the systemd service file.

| Variable | Default | Description |
|----------|---------|-------------|
| `MINECRAFT_MGMT_HOST` | `localhost` | Management API host |
| `MINECRAFT_MGMT_PORT` | `25566` | Management API port |
| `MINECRAFT_MGMT_SECRET` | (required) | 40-character authentication secret |
| `MINECRAFT_MGMT_TLS_ENABLED` | `true` | Enable TLS connection |
| `TEST_MODE` | `false` | When true, logs shutdown decisions without executing |
| `IDLE_TIMEOUT_MINUTES` | `10` | Minutes of no players before shutdown |
| `MIN_UPTIME_MINUTES` | `30` | Minimum server uptime before allowing shutdown |
| `POLL_INTERVAL_SECONDS` | `30` | Seconds between player checks |

## Testing

To test the watcher without actually shutting down your server, use **test mode**:

```bash
# Export configuration
export MINECRAFT_MGMT_HOST=localhost
export MINECRAFT_MGMT_PORT=25566
export MINECRAFT_MGMT_SECRET=your-40-character-secret-here
export TEST_MODE=true

# Optional: adjust timing for faster testing
export POLL_INTERVAL_SECONDS=10
export IDLE_TIMEOUT_MINUTES=2
export MIN_UPTIME_MINUTES=1

# Run the watcher
./minecraft-watcher
```

In test mode, the watcher will:
- Connect to the server and monitor players normally
- Log all player activity and idle times
- Display when shutdown conditions are met
- Log `TEST MODE: Would execute system shutdown: sudo systemctl poweroff` instead of actually shutting down
- Continue running after "shutdown" (won't exit)
- **Does NOT require sudo permissions** (safe to test as any user)

Example test mode output:
```
*** RUNNING IN TEST MODE - will not actually shut down server ***
Players online (0): []
No players online (idle for 2m30s)
Status: uptime=5m, idle=2m (thresholds: min_uptime=1m, idle_timeout=2m)
Shutdown conditions met: uptime=5m >= 1m AND idle=2m >= 2m
TEST MODE: Would execute system shutdown: sudo systemctl poweroff
```

### System Tests

Run automated system tests that execute the watcher binary and verify its behavior:

```bash
# Export configuration (same as above)
export MINECRAFT_MGMT_HOST=localhost
export MINECRAFT_MGMT_PORT=25566
export MINECRAFT_MGMT_SECRET=your-40-character-secret-here

# Run system tests
go test ./test -v
```

System tests verify the watcher:
- Builds successfully
- Runs in TEST_MODE
- Connects to Minecraft server
- Monitors player activity
- Logs shutdown decisions correctly

**Note:** System tests require a running Minecraft server with Management Protocol enabled. The watcher runs for ~15 seconds in test mode.

Example output:
```
=== RUN   TestWatcherConnection
--- PASS: TestWatcherConnection (15.24s)
    system_test.go:20: Building watcher...
    system_test.go:48: Starting watcher in test mode...
    system_test.go:60: watcher: minecraft-watcher starting...
    system_test.go:60: watcher: *** RUNNING IN TEST MODE - will not actually shut down server ***
    system_test.go:60: watcher: Successfully connected to Minecraft server
    system_test.go:60: watcher: Starting player monitoring loop...
    system_test.go:60: watcher: No players online (idle for 5s)
    system_test.go:60: watcher: TEST MODE: Would execute system shutdown: sudo systemctl poweroff
    system_test.go:92: ✓ Watcher running in TEST MODE
    system_test.go:98: ✓ Watcher successfully connected to Minecraft server
    system_test.go:104: ✓ Watcher successfully monitoring players
PASS
```

## Uninstallation

```bash
# Stop and disable the service
sudo systemctl stop minecraft-watcher
sudo systemctl disable minecraft-watcher

# Remove files
sudo rm /etc/systemd/system/minecraft-watcher.service
sudo rm /usr/local/bin/minecraft-watcher
sudo rm -rf /etc/minecraft-watcher

# Reload systemd
sudo systemctl daemon-reload
```

## Development

See [CLAUDE.md](CLAUDE.md) for project documentation and development guidelines.

## License

MIT License - see LICENSE file for details
