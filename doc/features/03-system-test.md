# Feature: System Test

## Status
- [x] Create test directory structure
- [x] Implement system test script
- [x] Test WebSocket connection
- [x] Test player count query
- [x] Validate JSON-RPC response parsing
- [x] Update README with test instructions

## Overview
Create a system test that validates the watcher can connect to a running Minecraft server and correctly query player information, without making assumptions about whether players are online.

## Requirements

### Test Assumptions
- Minecraft server is running
- Management Protocol is enabled
- Connection credentials are available via environment variables
- **No assumption** about player count (could be 0 or more)

### Test Validation
1. Successfully connect to WebSocket endpoint
2. Authenticate with bearer token
3. Send JSON-RPC request for player list
4. Receive valid response
5. Parse player data correctly
6. Log results (player count and names if any)

## Implementation Details

### Test Structure
- Location: `test/system_test.sh`
- Language: Bash script (for simplicity)
- Uses the built binary: `./minecraft-watcher`
- Runs with special environment variables

### Test Approach
Since we can't easily unit test Go WebSocket code in bash, we'll:
1. Create a minimal test mode that queries once and exits
2. Add `SINGLE_CHECK_MODE` environment variable
3. Run watcher with this mode
4. Check exit code and output for success

### Alternative: Go Test
- Location: `test/system_test.go`
- Uses same config loading and connection logic
- Performs single player check
- Reports results
- Better for CI/CD integration

## Testing Approach
```bash
# Run system test
export MINECRAFT_MGMT_SECRET=your-secret
./test/system_test.sh

# Expected output:
# ✓ Connected to server
# ✓ Queried player list
# ✓ Player count: N
# System test PASSED
```

## Implementation Choice
We'll implement a Go-based system test that can be run with `go test` for better integration and reliability.
