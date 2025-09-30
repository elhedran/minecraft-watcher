# Minecraft Server Auto-Shutdown Monitor

## Purpose
This repository implements a Linux background process that monitors a Minecraft server using the [Minecraft Server Management Protocol](https://minecraft.wiki/w/Minecraft_Server_Management_Protocol).

## Functionality
- Runs as a background service that starts on system boot
- Monitors the Minecraft server running on the same machine
- Implements auto-shutdown logic:
  - If the server has been running for more than 30 minutes
  - AND no user has been logged in for the last 10 minutes
  - THEN shut down the Minecraft server

## Implementation
- **Language**: Go - see [language choice rationale](doc/reference/language-choice.md)

## References
- [Language Choice](doc/reference/language-choice.md) - Why Go was selected
- [Minecraft Server Management Protocol](doc/reference/server-management-protocol.md) - Local reference documentation
- [Official Protocol Documentation](https://minecraft.wiki/w/Minecraft_Server_Management_Protocol) - Original source

## Ways of Working

### Feature Development Process
1. **Keep features minimal** - Each feature should be scoped to the smallest useful increment
2. **Plan before implementing** - Before coding any feature:
   - Create an implementation plan document under `doc/features/`
   - Document the approach, components, and steps required
3. **Track progress** - As the feature is implemented:
   - Update the feature document to indicate completed parts
   - Mark implementation status for each component/step