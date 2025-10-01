# Feature: Server Connectivity Check

## Status
- [x] Configuration loading
- [x] WebSocket connection establishment
- [x] Authentication with bearer token
- [x] Connection retry loop with backoff
- [x] Logging for connection attempts

## Overview
The service must establish and maintain connectivity to the Minecraft server's Management Protocol endpoint before performing any monitoring functions.

## Requirements
1. Load configuration from environment variables
2. Attempt to connect to the WebSocket endpoint
3. Authenticate using the bearer token
4. Retry on connection failure with exponential backoff
5. Log connection status (attempts, success, failures)
6. Block until successful connection established

## Implementation Details

### Configuration
Read from environment variables:
- `MINECRAFT_MGMT_HOST` (default: "localhost")
- `MINECRAFT_MGMT_PORT` (default: "25566")
- `MINECRAFT_MGMT_SECRET` (required)
- `MINECRAFT_MGMT_TLS_ENABLED` (default: "true")

### Connection Logic
1. Build WebSocket URL: `ws://` or `wss://` based on TLS setting
2. Create HTTP header with `Authorization: Bearer <secret>`
3. Attempt WebSocket dial
4. On failure: wait with exponential backoff (1s, 2s, 4s, 8s, max 30s)
5. On success: log and proceed

### Error Handling
- Invalid/missing configuration → log error and exit
- Connection refused → retry with backoff
- Authentication failure (401) → log error and exit
- Network errors → retry with backoff

### Dependencies
- WebSocket library: Use standard library `golang.org/x/net/websocket` or external like `github.com/gorilla/websocket`
- Logging: Standard library `log` package

## Testing Approach
- Manual testing with Minecraft server running
- Manual testing with server offline (verify retry behavior)
- Manual testing with incorrect secret (verify auth failure handling)
