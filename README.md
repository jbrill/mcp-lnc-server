# MCP LNC Server (Read-Only)

MCP LNC Server is a **read-only** Model Context Protocol (MCP) server that provides secure access to Lightning Network nodes through Lightning Node Connect (LNC). It enables AI assistants like Claude to safely query Lightning Network data without performing any state-changing operations.

## Architecture

The server follows a modular, service-oriented architecture with separate service packages for read-only operations:

- **ConnectionService**: Manages LNC connections and authentication  
- **NodeService**: Queries node information and balance data
- **InvoiceService**: Decodes BOLT11 invoices and lists created invoices
- **PaymentService**: Lists payment history and tracks payment status  
- **ChannelService**: Lists Lightning channels and pending channel states
- **PeerService**: Lists peers and queries network graph information
- **OnChainService**: Lists UTXOs, transaction history, and fee estimates

Each service is designed to be independent, testable, and extensible with clear interfaces and dependency injection. **All operations are read-only** - no payments, channel operations, or other state changes are supported.

## Installation

### Prerequisites

- Go 1.24.4 or higher (with toolchain downloads enabled)
- LND v0.19.x with Lightning Node Connect enabled
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

#### Read-Only Design  

This server provides **only read-only tools** for safely exploring Lightning Network data. All write operations (payments, channel operations, address generation, etc.) have been removed to ensure the server cannot modify node state or funds.

## Available Tools (Read-Only)

### Connection Management
- `lnc_connect`: Connect to Lightning node via LNC (requires `pairingPhrase`, `password`)
- `lnc_disconnect`: Disconnect from current node

### Node Information
- `lnc_get_info`: Get comprehensive node information
- `lnc_get_balance`: Get wallet and channel balances

### Invoice Management (Read-Only)
- `lnc_decode_invoice`: Decode BOLT11 invoice (requires `invoice`)
- `lnc_list_invoices`: List all invoices created by this node
- `lnc_lookup_invoice`: Look up specific invoice by payment hash

### Payment History (Read-Only)
- `lnc_list_payments`: List historical payments made by this node
- `lnc_track_payment`: Track the status of a specific payment by hash

### Channel Information (Read-Only)
- `lnc_list_channels`: List all channels with detailed information
- `lnc_pending_channels`: List pending channels in various states

### Peer and Network Information (Read-Only) 
- `lnc_list_peers`: List connected peers with connection details
- `lnc_describe_graph`: Get Lightning Network graph information
- `lnc_get_node_info`: Get detailed info about a specific node (requires `pub_key`)

### On-Chain Wallet Information (Read-Only)
- `lnc_list_unspent`: List unspent transaction outputs (UTXOs)
- `lnc_get_transactions`: Get on-chain transaction history
- `lnc_estimate_fee`: Estimate transaction fees for different confirmation targets

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

// Decode an invoice
{
  "tool": "lnc_decode_invoice",
  "arguments": {
    "invoice": "lnbc10u1p3..."
  }
}

// List payment history  
{
  "tool": "lnc_list_payments",
  "arguments": {
    "include_incomplete": true,
    "max_payments": 50
  }
}
```

### Channel Information

```json
// List channels
{
  "tool": "lnc_list_channels",
  "arguments": {
    "active_only": true
  }
}

// Get pending channels
{
  "tool": "lnc_pending_channels"
}
```

## Development

### Project Structure

```
mcp-lnc-server/
├── server.go                 # Main MCP server entry point
├── daemon.go                 # Daemon management and lifecycle
├── tools/                    # Lightning Network tool implementations (read-only)
│   ├── connection.go         # LNC connection management
│   ├── node.go              # Node information and balance queries  
│   ├── invoices.go          # Invoice decoding and listing
│   ├── payments.go          # Payment history and tracking
│   ├── channels.go          # Channel information queries
│   ├── peers.go             # Peer information and network graph
│   └── onchain.go           # On-chain wallet information
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

# Test inside Docker (Go 1.24.5)
make test-docker

# Build Docker image for deployment
docker build -t mcp-lnc-server .

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o mcp-lnc-server-linux
GOOS=darwin GOARCH=amd64 go build -o mcp-lnc-server-macos
GOOS=windows GOARCH=amd64 go build -o mcp-lnc-server-windows.exe
```

### Local Integration Testing

1. Start an LND v0.19.x node in regtest with Lightning Node Connect enabled
   (`lnd --noseedbackup --pilot=lightninglabs-remote ...`).
2. Generate a pairing phrase with the Lightning Labs `zanelit` helper:
   ```bash
   zanelit sessions add --label dev-session \
     --mailboxserveraddr aperture:11110 --type admin --devserver
   ```
3. Export development environment variables:
   ```bash
   export LNC_MAILBOX_SERVER="aperture:11110"
   export LNC_DEV_MODE="true"
   export LNC_INSECURE="true"
   ```
4. Build and run the MCP server locally:
   ```bash
   go build -o mcp-lnc-server
   ./mcp-lnc-server
   ```
5. Connect via Claude Desktop or another MCP client and exercise read-only
   tools (`lnc_get_info`, `lnc_list_channels`, `lnc_list_unspent`, `lnc_list_payments`).
6. Test invoice decoding and payment history tools to verify Lightning Network
   data can be queried safely without any risk of state changes.
7. If your local Go version lags behind the required toolchain, use Docker to
   run the tests without installing Go:
   ```bash
   make test-docker
   ```

### Running with Docker

After building the image, start the server in a container:

```bash
docker run --rm \
  -e LNC_MAILBOX_SERVER="mailbox.terminal.lightning.today:443" \
  -e LNC_DEV_MODE="false" \
  -e LNC_INSECURE="false" \
  mcp-lnc-server
```

Swap environment variables for regtest/local deployments as needed. The MCP
client (Claude, Cursor, etc.) will still prompt for the pairing phrase and
password via the `lnc_connect` tool.

## Security

- **Read-only design** eliminates risk of accidental payments or state changes
- All connections use LNC's built-in TLS encryption and authentication
- No sensitive data is logged or stored permanently  
- Pairing phrases are handled securely in memory only
- Direct node-to-node communication through encrypted tunnels
- Use `devMode` and `insecure` only for local development

## License

MIT License - see LICENSE file for details
