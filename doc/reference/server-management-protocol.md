# Minecraft Server Management Protocol

## Overview
The Minecraft Server Management Protocol is a WebSocket-based API using JSON-RPC 2.0 specification for managing and monitoring Minecraft servers.

## Connection Details
- **Protocol**: WebSocket
- **Endpoint**: `ws://<management-server-host>:<management-server-port>` (or `wss://` if TLS enabled)
- **Authentication**: Bearer token via `Authorization` header
- **Data Format**: JSON-RPC 2.0

## Server Configuration (server.properties)

| Property | Default | Description |
|----------|---------|-------------|
| `management-server-enabled` | `false` | Set to true to enable the API |
| `management-server-host` | `localhost` | Host of the API endpoint |
| `management-server-port` | `0` | Port of the API endpoint (0 = random port on startup) |
| `management-server-secret` | Auto-generated | 40 alphanumeric characters for authentication |
| `management-server-tls-enabled` | `true` | Enable/disable TLS |
| `management-server-tls-keystore` | None | Path to PKCS12 keystore file |
| `management-server-tls-keystore-password` | None | Keystore password |

## Key Methods for Monitoring

### Players (`minecraft:players`)
- **GET `/`** - Get all connected players
  - Parameters: None
  - Result: `players: Array<Player>`

- **POST `/kick`** - Kick players
  - Parameters: `kick: Array<Kick Player>`
  - Result: `kicked: Array<Player>`

### Server (`minecraft:server`)
- **GET `/status`** - Get server status
  - Parameters: None
  - Result: `status: Server State`

- **POST `/save`** - Save server state
  - Parameters: `flush: boolean`
  - Result: `saving: boolean`

- **POST `/stop`** - Stop server
  - Parameters: None
  - Result: `stopping: boolean`

### Server Settings (`minecraft:serversettings`)
- **GET `/autosave`** - Check if automatic world saving is enabled
  - Parameters: None
  - Result: `enabled: boolean`

## Key Notifications for Monitoring

### Player Notifications (`minecraft:notification/players`)
- **`/joined`** - Player joined server
  - Parameters: `player: Player`

- **`/left`** - Player left server
  - Parameters: `player: Player`

### Server Notifications (`minecraft:notification/server`)
- **`/started`** - Server started
- **`/stopping`** - Server shutting down
- **`/saving`** - Server save started
- **`/saved`** - Server save completed
- **`/status`** - Server status heartbeat
  - Parameters: `status: Server State`

## Data Schemas

### Player
```json
{
  "name": "string",
  "id": "string"
}
```

### Server State
```json
{
  "players": ["Array<Player>"],
  "started": "boolean",
  "version": {
    "protocol": "integer",
    "name": "string"
  }
}
```

## Example API Calls

### Get Connected Players
**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:players/",
  "id": 1
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "players": [
      {"id": "853c80ef-3c37-49fd-aa49-938b674adae6", "name": "jeb_"}
    ]
  }
}
```

### Get Server Status
**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:server/status",
  "id": 2
}
```

### Stop Server
**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:server/stop",
  "id": 3
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "stopping": true
  }
}
```

## Example Notifications

### Player Joined
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:notification/players/joined",
  "params": [
    {"id": "853c80ef-3c37-49fd-aa49-938b674adae6", "name": "jeb_"}
  ]
}
```

### Player Left
```json
{
  "jsonrpc": "2.0",
  "method": "minecraft:notification/players/left",
  "params": [
    {"id": "853c80ef-3c37-49fd-aa49-938b674adae6", "name": "jeb_"}
  ]
}
```

## Error Handling

Errors follow JSON-RPC 2.0 specification:

**Error Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": "Method not found: minecraft:foo/bar"
  }
}
```

**Common Error Codes:**
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `401`: Unauthorized (invalid/missing bearer token)

## API Discovery

Call `rpc.discover` to get the full API schema:

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "rpc.discover",
  "id": 1
}
```

This returns all supported methods and notifications for the running server.

## Implementation Notes for Auto-Shutdown Monitor

For our use case (monitoring server uptime and player activity), we need to:

1. **Connect** to the WebSocket endpoint with bearer token authentication
2. **Subscribe** to player join/leave notifications:
   - `minecraft:notification/players/joined`
   - `minecraft:notification/players/left`
3. **Poll** server status periodically:
   - `minecraft:server/status` - to check uptime and current players
4. **Trigger shutdown** when conditions are met:
   - `minecraft:server/stop` - to shut down the server

## Source
Original documentation: https://minecraft.wiki/w/Minecraft_Server_Management_Protocol
