# MCP LNC Server Architecture

This document captures the high-level structure of the MCP LNC Server as it moves toward production readiness.

## Layered Overview

- **MCP Frontend**: `server.Server` hosts an `MCPServer` instance that is reachable via stdio by Claude Desktop or other MCP clients. All tool registration flows through this boundary.
- **Service Manager**: `internal/services.Manager` owns the Lightning RPC clients. It wires MCP tools to service handlers and now enforces a read-only default toolset unless `LNC_ALLOW_MUTATING_TOOLS=true` is provided.
- **Service Packages**: Each domain (connection, node, invoices, channels, payments, peers, on-chain) lives under `tools/` with a single responsibility interface. Handlers translate from MCP JSON arguments to LND gRPC calls.
- **LNC Connection Layer**: `tools.ConnectionService` pairs with Lightning Node Connect, establishes the mailbox tunnel, and hands the resulting `grpc.ClientConn` to the manager callback.
- **Shared Infrastructure**: Configuration (`internal/config`), structured logging (`internal/logging`), error types (`internal/errors`), and request-scoped context helpers (`internal/context`) provide cross-cutting concerns.

## Tool Registration Flow

```
server.NewServer(cfg, logger)
  └─ services.NewManager(logger, cfg.AllowMutatingTools)
       └─ InitializeServices()
       └─ RegisterTools(mcpServer)
            ├─ Connection tools (always)
            ├─ Read-only domain tools (always)
            └─ Mutating tools (only when allowed)
```

## Tool Modes

| Mode       | Description                                               | Typical Use |
|------------|-----------------------------------------------------------|-------------|
| Read-only  | Default. Exposes inspection tools only.                   | First-run reviews, demo nodes |
| Mutating   | Enabled via `LNC_ALLOW_MUTATING_TOOLS=true`. Adds payment, channel, on-chain write operations. | Controlled integration tests, trusted environments |

## Configuration Surface

Environment variables provide the main tuning mechanism:

- `LNC_ALLOW_MUTATING_TOOLS` governs tool mode.
- `LNC_MAILBOX_SERVER`, `LNC_DEV_MODE`, `LNC_INSECURE` describe how to reach the mailbox.
- `LNC_CONNECT_TIMEOUT`, `LNC_MAX_RETRIES` define connection resilience.

Future roadmap items include richer version negotiation for LND and multi-session management, but the current codebase is intentionally conservative: new operators start in a safe read-only state with explicit steps to unlock write capabilities.
