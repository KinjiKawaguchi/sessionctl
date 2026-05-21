# sessionctl

MCP server for persistent SSH/Telnet session management with expect/send-based interactive CLI control and multi-hop session chaining.

## Install

```bash
npx -y sessionctl
```

Or with Go:

```bash
go install github.com/KinjiKawaguchi/sessionctl/cmd/sessionctl@latest
```

## MCP Configuration

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "sessionctl": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "sessionctl"]
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `session_open` | Open SSH/Telnet session with authentication |
| `session_exec` | Execute command, wait for prompt, return output |
| `session_interact` | Send input (password, confirmation) and optionally wait for pattern |
| `session_chain` | Start sub-session within existing session (e.g., telnet from SSH) |
| `session_list` | List all active sessions |
| `session_close` | Close session (sends exit for chained sessions) |

## Device Profiles

Built-in profiles: `cisco_ios`, `yamaha_rtx`, `unix`. Specify via `profile` parameter in `session_open`.

## Example: Multi-hop to Cisco ASA

```
session_open(host="10.1.11.253", username="admin", password="secret", profile="cisco_ios")
  → session_chain(command="telnet 10.1.31.251")
    → session_interact(input="admin", expect="Password:")
    → session_interact(input="secret", expect="ciscoasa>")
    → session_exec(command="show version")
```

## Architecture

Data-Oriented Design. No interfaces — tagged unions with `Kind` field + switch.

```
internal/
├── expect/     # Expect engine: CircularBuffer + Pattern + Engine
├── session/    # Session store: flat slices + index maps
├── device/     # Device profiles: ProfileTable
├── transport/  # SSH/Telnet: Transport tagged union
└── mcp/        # MCP tool handlers
```

## Development

```bash
# Build
go build -o ./bin/sessionctl ./cmd/sessionctl/

# Unit tests
go test ./internal/... -race

# Integration tests (in-process fake devices)
go test ./test/integration/ -tags=integration

# Run fake devices for manual testing
go run ./cmd/fakedevices/
```

## License

MIT
