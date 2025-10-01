# Feature: Player Monitoring and Auto-Shutdown

## Status
- [x] Add test mode configuration
- [x] Implement JSON-RPC 2.0 client for API calls
- [x] Query player list periodically
- [x] Track server uptime
- [x] Track last player activity time
- [x] Implement shutdown decision logic
- [x] Execute shutdown command (or log in test mode)
- [x] Update README with test mode documentation

## Overview
Monitor connected players and automatically shut down the server when idle conditions are met.

## Requirements

### Shutdown Conditions
1. Server has been running for more than 30 minutes (configurable)
2. No players have been online for the last 10 minutes (configurable)
3. When both conditions are met → trigger shutdown

### Test Mode
- Environment variable: `TEST_MODE=true`
- When enabled: Log shutdown decision instead of executing
- Default: `false` (production mode)

### Monitoring Loop
- Poll interval: Every 30 seconds (configurable)
- Call: `minecraft:players/` to get connected players
- Track: Last time any player was online
- Track: Server start time (for uptime check)

## Implementation Details

### Configuration
Add environment variables:
- `TEST_MODE` (default: "false")
- `IDLE_TIMEOUT_MINUTES` (default: "10")
- `MIN_UPTIME_MINUTES` (default: "30")
- `POLL_INTERVAL_SECONDS` (default: "30")

### JSON-RPC 2.0 Protocol
Request format:
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:players/",
  "id": 1
}
```

Response format:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "players": [
      {"id": "uuid", "name": "playername"}
    ]
  }
}
```

### Shutdown Logic
1. Every poll interval:
   - Send JSON-RPC request for player list
   - If players.length > 0: Update lastPlayerTime to now
   - Check if uptime > MIN_UPTIME_MINUTES
   - Check if (now - lastPlayerTime) > IDLE_TIMEOUT_MINUTES
   - If both true: Execute shutdown

2. Shutdown execution:
   - Test mode: Log "Would shut down server now"
   - Production: Send `minecraft:server/stop` JSON-RPC request

### Error Handling
- WebSocket disconnection → reconnect with retry logic
- JSON-RPC errors → log and continue
- Invalid responses → log and continue

## Testing Approach
- Run with `TEST_MODE=true` and monitor logs
- Verify player detection works
- Verify idle timeout calculation
- Verify uptime check works
- Verify "would shutdown" log appears at correct time
