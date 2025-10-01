# Feature: System Shutdown Instead of Server Shutdown

## Status
- [x] Update shutdown logic to execute system shutdown command
- [x] Add proper permissions documentation for shutdown capability
- [x] Update systemd service configuration
- [x] Test in test mode (dry-run)
- [x] Update README with deployment requirements
- [x] Document security considerations

## Overview
Change the watcher to shut down the entire Linux instance when idle conditions are met, rather than just stopping the Minecraft server. This saves cloud costs by stopping the EC2 instance completely.

## Current Behavior
When idle conditions are met:
- Production mode: Sends `minecraft:server/stop` JSON-RPC command
- Test mode: Logs "Would execute server shutdown now"

## New Behavior
When idle conditions are met:
- Production mode: Executes `sudo shutdown -h now` (or `sudo poweroff`)
- Test mode: Logs "Would execute system shutdown: sudo shutdown -h now"

## Implementation Details

### Shutdown Command Options
1. **`sudo shutdown -h now`** - Standard, logs shutdown, allows services to cleanup
2. **`sudo poweroff`** - Immediate shutdown
3. **`sudo systemctl poweroff`** - systemd-based shutdown (recommended)

**Recommended:** `sudo systemctl poweroff` for clean systemd integration

### Execution Method
Use Go's `os/exec` package:
```go
cmd := exec.Command("sudo", "systemctl", "poweroff")
err := cmd.Run()
```

### Permissions Required
The `minecraft` user (or whatever user runs the service) needs passwordless sudo for shutdown:

**Add to `/etc/sudoers.d/minecraft-watcher`:**
```
minecraft ALL=(ALL) NOPASSWD: /usr/bin/systemctl poweroff
minecraft ALL=(ALL) NOPASSWD: /usr/bin/shutdown
```

Or if running as a system service with limited scope:
```
minecraft ALL=(ALL) NOPASSWD: /bin/systemctl poweroff
```

### Code Changes

#### 1. Update shutdownServer function
Rename to `shutdownSystem` and change implementation:
```go
func shutdownSystem(testMode bool) error {
    if testMode {
        log.Println("TEST MODE: Would execute system shutdown: sudo systemctl poweroff")
        return nil
    }

    log.Println("Idle timeout reached. Shutting down system...")
    cmd := exec.Command("sudo", "systemctl", "poweroff")

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to shutdown system: %w", err)
    }

    log.Println("System shutdown command executed")
    return nil
}
```

#### 2. Remove WebSocket shutdown call
No longer need to call `minecraft:server/stop` since the entire system is shutting down.

#### 3. Update logging
Change log messages to reflect system shutdown instead of server shutdown.

### Systemd Service Updates
No changes required to the service file, but documentation should note:
- Service will stop when system shuts down (normal behavior)
- Service should be enabled to start on boot: `systemctl enable minecraft-watcher`

## Security Considerations

### Pros
- Watcher has minimal permissions (only shutdown capability)
- Shutdown command is explicitly whitelisted in sudoers
- No shell command injection risk (using exec.Command with separate args)

### Cons
- Watcher can shut down the entire system
- Must trust the watcher logic to not shutdown inappropriately
- Test mode is critical for safe testing

### Mitigation
- Keep test mode prominent in logging
- Require explicit configuration (environment variable)
- Log all shutdown decisions with full context
- Use sudo restrictions to limit to specific shutdown commands only

## Testing Approach

### Test Mode Testing
```bash
export TEST_MODE=true
export MIN_UPTIME_MINUTES=1
export IDLE_TIMEOUT_MINUTES=1
export POLL_INTERVAL_SECONDS=10
./minecraft-watcher
```

Should log: `TEST MODE: Would execute system shutdown: sudo systemctl poweroff`

### Permission Testing
Before deploying to production:
1. Create sudoers entry
2. Test with: `sudo -u minecraft sudo systemctl poweroff` (in a non-critical environment)
3. Verify command succeeds

### Production Testing
- Deploy to a non-critical test EC2 instance first
- Monitor logs during shutdown
- Verify instance stops cleanly
- Verify instance restarts correctly when manually started

## Documentation Updates

### README.md
Add section under Installation:
- **System Shutdown Permissions** - sudoers configuration
- Security note about shutdown capability
- Warning about test mode importance

### Systemd Service
No changes needed, but document:
- Service stops automatically on system shutdown
- Restart policy not relevant (system is shutting down)

## Configuration
No new environment variables needed. Existing configuration applies:
- `TEST_MODE` - critical for testing
- `MIN_UPTIME_MINUTES` - minimum uptime before allowing shutdown
- `IDLE_TIMEOUT_MINUTES` - idle time before shutdown
- `POLL_INTERVAL_SECONDS` - how often to check

## Rollback Plan
If issues occur:
1. Stop service: `sudo systemctl stop minecraft-watcher`
2. Disable service: `sudo systemctl disable minecraft-watcher`
3. Remove from boot if needed
4. Revert to server-only shutdown if preferred
