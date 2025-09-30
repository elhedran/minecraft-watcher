# Language Choice: Go

## Decision
The minecraft-watcher project will be implemented in **Go (Golang)**.

## Reasoning

### No Separate Runtime Required
Go compiles to a single static binary with no runtime dependencies. This means:
- No need to install Python, Node.js, or any other interpreter on the target system
- Simplified deployment - just copy the binary to the server
- No version compatibility issues with runtime environments
- Reduces attack surface by minimizing installed software

### Low Resource Footprint
- Compiled Go binaries are lightweight (~5-10MB typical memory usage)
- Efficient for a background monitoring process
- Minimal CPU usage when idle
- Important for a daemon that runs continuously on a Minecraft server

### Excellent WebSocket and JSON Support
- Strong WebSocket libraries available (`gorilla/websocket`, `nhooyr.io/websocket`)
- Built-in JSON marshaling/unmarshaling in standard library
- JSON-RPC 2.0 client libraries available
- Well-suited for the Minecraft Server Management Protocol requirements

### Native Concurrency
- Goroutines enable efficient handling of:
  - WebSocket connection management
  - Notification listeners
  - Periodic status polling
  - Shutdown timer logic
- Clean concurrent code without complex threading

### Robust Error Handling
- Explicit error handling encourages defensive programming
- Critical for a system service that must handle:
  - Network interruptions
  - Server restarts
  - Authentication failures
  - WebSocket reconnection

### Strong Standard Library
- Built-in support for:
  - HTTP/HTTPS clients
  - TLS certificate handling
  - Time management and timers
  - Signal handling for daemon control
- Reduces external dependencies

### Linux Daemon Integration
- Easy integration with systemd
- Standard signal handling (SIGTERM, SIGINT)
- Simple to configure as a system service
- Clean shutdown and restart behavior

### Maintainability
- Static typing catches errors at compile time
- Clear, readable syntax
- Good tooling (go fmt, go vet, go test)
- Strong community and documentation

## Alternative Considerations

**Python** was considered but rejected due to:
- Runtime dependency requirement
- Higher memory footprint
- Need for virtual environment management

**Bash scripting** was considered but rejected due to:
- Complex WebSocket authentication handling
- Fragile state management
- Difficult error handling and recovery

## Implementation Approach
- Use standard Go project layout
- Leverage `gorilla/websocket` or `nhooyr.io/websocket` for WebSocket client
- Implement as a systemd service
- Support configuration via environment variables or config file
- Include proper logging for monitoring and debugging
