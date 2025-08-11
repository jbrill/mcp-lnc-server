# MCP LNC Server

MCP LNC Server is a Model Context Protocol (MCP) server that provides secure access to Lightning Network nodes through Lightning Node Connect (LNC). It enables AI assistants like Claude to interact with Lightning Network nodes using a comprehensive set of tools for node management, payments, channels, and on-chain operations.

## Architecture

The server follows a modular, service-oriented architecture with separate service packages:

- **ConnectionService**: Manages LNC connections and authentication
- **NodeService**: Handles node information queries and balance checks
- **InvoiceService**: Manages invoice creation and BOLT11 decoding
- **PaymentService**: Handles both invoice payments and keysend transactions
- **ChannelService**: Manages Lightning channel operations (open, close, list)
- **PeerService**: Handles peer connections and network graph queries
- **OnChainService**: Manages on-chain wallet operations and transactions

Each service is designed to be independent, testable, and extensible with clear interfaces and dependency injection.

## Installation

### Prerequisites

- Go 1.21 or higher
- A Lightning Network node (LND, Core Lightning, etc.)
- LNC pairing phrase and password

### Build

```bash
git clone https://github.com/jbrill/mcp-lnc-server
cd mcp-lnc-server/mcp-lnc-server
go build -o mcp-lnc-server
```

## Configuration

### Claude Desktop Setup

Add the following to your Claude desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcp-servers": {
    "lnc": {
      "command": "/path/to/mcp-lnc-server",
      "type": "stdio"
    }
  }
}
```

### Environment Variables

Configure default connection parameters:

```bash
# Production settings
export LNC_MAILBOX_SERVER="mailbox.terminal.lightning.today:443"

# Development/regtest settings
export LNC_MAILBOX_SERVER="aperture:11110"
export LNC_DEV_MODE="true"
export LNC_INSECURE="true"

# Connection settings
export LNC_CONNECT_TIMEOUT="30"
export LNC_MAX_RETRIES="3"
```

## Available Tools

### Connection Management
- `lnc_connect`: Connect to Lightning node via LNC (requires `pairingPhrase`, `password`)
- `lnc_disconnect`: Disconnect from current node

### Node Information
- `lnc_get_info`: Get comprehensive node information
- `lnc_get_balance`: Get wallet and channel balances

### Invoice Management
- `lnc_create_invoice`: Create Lightning invoice (requires `amount` in sats)
- `lnc_decode_invoice`: Decode BOLT11 invoice (requires `invoice`)

### Payments
- `lnc_pay_invoice`: Pay BOLT11 invoice (requires `invoice`)
- `lnc_send_payment`: Send keysend payment (requires `destination`, `amount_sats`)

### Channel Management
- `lnc_list_channels`: List all channels
- `lnc_open_channel`: Open new channel (requires `node_pubkey`, `local_funding_amount`)
- `lnc_close_channel`: Close channel (requires `channel_point`)
- `lnc_pending_channels`: List pending channels

### Peer Management
- `lnc_list_peers`: List connected peers
- `lnc_connect_peer`: Connect to peer (requires `node_pubkey`, `address`)
- `lnc_disconnect_peer`: Disconnect from peer (requires `node_pubkey`)
- `lnc_describe_graph`: Get network graph
- `lnc_get_node_info`: Get specific node info (requires `pub_key`)

### On-Chain Wallet
- `lnc_new_address`: Generate new address
- `lnc_send_coins`: Send on-chain transaction (requires `address`, `amount`)
- `lnc_list_unspent`: List UTXOs
- `lnc_get_transactions`: Get transaction history
- `lnc_estimate_fee`: Estimate transaction fees

## Usage Examples

### Basic Operations

```json
// Connect to node
{
  "tool": "lnc_connect",
  "arguments": {
    "pairingPhrase": "your ten word pairing phrase here",
    "password": "your_password"
  }
}

// Get node information
{
  "tool": "lnc_get_info"
}

// Create invoice
{
  "tool": "lnc_create_invoice",
  "arguments": {
    "amount": 1000,
    "memo": "Payment for services"
  }
}

// Pay invoice
{
  "tool": "lnc_pay_invoice",
  "arguments": {
    "invoice": "lnbc10u1p3...",
    "max_fee_sats": 1000
  }
}
```

### Channel Operations

```json
// Open channel
{
  "tool": "lnc_open_channel",
  "arguments": {
    "node_pubkey": "03abc123...",
    "local_funding_amount": 100000,
    "push_sat": 0,
    "private": false
  }
}

// List channels
{
  "tool": "lnc_list_channels",
  "arguments": {
    "active_only": true
  }
}
```

## Development

### Project Structure

```
mcp-lnc-server/
├── server.go                 # Main MCP server entry point
├── daemon.go                 # Daemon management and lifecycle
├── tools/                    # Lightning Network tool implementations
│   ├── connection.go         # LNC connection management
│   ├── node.go              # Node information and balance queries
│   ├── invoices.go          # Invoice creation and decoding
│   ├── payments.go          # Payment sending and processing
│   ├── channels.go          # Channel management operations
│   ├── peers.go             # Peer management and network graph
│   └── onchain.go           # On-chain wallet operations
├── internal/                 # Internal packages
│   ├── config/              # Configuration management
│   ├── logging/             # Structured logging
│   ├── errors/              # Error handling and types
│   ├── interfaces/          # Service interfaces
│   ├── client/              # Lightning client wrappers
│   └── services/            # Service management
└── README.md                # This file
```

### Building and Testing

```bash
# Build
go build -o mcp-lnc-server

# Test
go test ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o mcp-lnc-server-linux
GOOS=darwin GOARCH=amd64 go build -o mcp-lnc-server-macos
GOOS=windows GOARCH=amd64 go build -o mcp-lnc-server-windows.exe
```

### Development Environment

For local development with regtest:

1. Set up LND in regtest mode
2. Generate LNC pairing phrase:
   ```bash
   zanelit sessions add --label dev-session --mailboxserveraddr aperture:11110 --type admin --devserver
   ```
3. Configure environment:
   ```bash
   export LNC_MAILBOX_SERVER="aperture:11110"
   export LNC_DEV_MODE="true"
   export LNC_INSECURE="true"
   ```

## Security

- All connections use LNC's built-in TLS encryption and authentication
- No sensitive data is logged or stored permanently
- Pairing phrases are handled securely in memory only
- Direct node-to-node communication through encrypted tunnels
- Use `devMode` and `insecure` only for local development

## License

MIT License - see LICENSE file for details
