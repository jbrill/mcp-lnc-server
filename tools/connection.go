// Package tools provides MCP tool implementations for Lightning Network operations.
//
// This package contains all the MCP tools that allow AI assistants to interact
// with Lightning Network nodes through various service interfaces.
package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightninglabs/lightning-node-connect/mailbox"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ConnectionService handles LNC connection management.
type ConnectionService struct {
	Connection         *grpc.ClientConn
	ConnectionCallback func(*grpc.ClientConn)
}

// NewConnectionService creates a new connection service.
func NewConnectionService(
	callback func(*grpc.ClientConn)) *ConnectionService {
	return &ConnectionService{
		ConnectionCallback: callback,
	}
}

// ConnectTool returns the MCP tool definition for connecting to LNC.
func (s *ConnectionService) ConnectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_connect",
		Description: "Connect to a Lightning node using LNC pairing phrase",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"pairingPhrase": map[string]any{
					"type":        "string",
					"description": "The LNC pairing phrase (10 words)",
				},
				"password": map[string]any{
					"type":        "string",
					"description": "The LNC password",
				},
				"mailbox": map[string]any{
					"type": "string",
					"description": "Custom mailbox server address " +
						"(optional, e.g., 'localhost:11110' for regtest)",
				},
				"devMode": map[string]any{
					"type":        "boolean",
					"description": "Enable dev mode for local/regtest environments (optional)",
				},
				"insecure": map[string]any{
					"type":        "boolean",
					"description": "Skip TLS verification for dev environments (optional)",
				},
			},
			Required: []string{"pairingPhrase", "password"},
		},
	}
}

// HandleConnect handles the LNC connection request.
func (s *ConnectionService) HandleConnect(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pairingPhrase, ok := request.Params.Arguments["pairingPhrase"].(string)
	if !ok {
		return mcp.NewToolResultError("pairingPhrase is required"), nil
	}

	password, ok := request.Params.Arguments["password"].(string)
	if !ok {
		return mcp.NewToolResultError("password is required"), nil
	}

	// Validate pairing phrase format
	words := strings.Split(strings.TrimSpace(pairingPhrase), " ")
	if len(words) != 10 {
		return mcp.NewToolResultError(
			"pairingPhrase must contain exactly 10 words"), nil
	}

	// Get connection parameters with environment variable defaults
	mailboxServer := getMailboxServer(request.Params.Arguments)
	if mailboxServer == "" {
		if envMailbox := os.Getenv("LNC_MAILBOX_SERVER"); envMailbox != "" {
			mailboxServer = envMailbox
		} else {
			mailboxServer = "mailbox.terminal.lightning.today:443"
		}
	}

	// Check for dev mode with environment variable default
	devMode := false
	if dev, ok := request.Params.Arguments["devMode"].(bool); ok {
		devMode = dev
	} else if envDev := os.Getenv("LNC_DEV_MODE"); envDev != "" {
		devMode, _ = strconv.ParseBool(envDev)
	}

	// Check for insecure mode with environment variable default
	insecure := false
	if ins, ok := request.Params.Arguments["insecure"].(bool); ok {
		insecure = ins
	} else if envInsecure := os.Getenv("LNC_INSECURE"); envInsecure != "" {
		insecure, _ = strconv.ParseBool(envInsecure)
	}

	// Get timeout from environment or use default
	timeout := 30 * time.Second
	if envTimeout := os.Getenv("LNC_CONNECT_TIMEOUT"); envTimeout != "" {
		if seconds, err := strconv.Atoi(envTimeout); err == nil {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	// Create connection context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Log connection attempt
	zap.L().Info("Attempting LNC connection",
		zap.String("mailbox", mailboxServer),
		zap.Bool("devMode", devMode),
		zap.Bool("insecure", insecure),
	)

	// Establish LNC connection
	conn, nodeInfo, err := s.connectToLNC(connectCtx, pairingPhrase,
		password, mailboxServer, devMode, insecure)
	if err != nil {
		zap.L().Error("LNC connection failed", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to connect to Lightning node: %v", err)), nil
	}

	// Store connection
	s.Connection = conn

	// Notify main server of new connection
	if s.ConnectionCallback != nil {
		s.ConnectionCallback(conn)
	}

	// Return success response
	return mcp.NewToolResultText(fmt.Sprintf(`{
		"connected": true,
		"node_pubkey": "%s",
		"alias": "%s",
		"num_channels": %d,
		"num_peers": %d,
		"version": "%s",
		"mailbox_server": "%s"
	}`, nodeInfo.IdentityPubkey, nodeInfo.Alias, nodeInfo.NumActiveChannels,
		nodeInfo.NumPeers, nodeInfo.Version, mailboxServer)), nil
}

// connectToLNC establishes the actual LNC connection.
func (s *ConnectionService) connectToLNC(ctx context.Context,
	pairingPhrase, password, mailboxServer string, devMode,
	insecure bool) (*grpc.ClientConn, *lnrpc.GetInfoResponse, error) {
	zap.L().Debug("Starting LNC connection process",
		zap.String("mailbox", mailboxServer),
		zap.Int("pairing_phrase_words", len(strings.Split(pairingPhrase, " "))),
		zap.Bool("dev_mode", devMode),
		zap.Bool("insecure", insecure),
		zap.Bool("has_password", password != ""),
	)

	// Generate a new private key for this session
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	zap.L().Debug("Generated session private key")

	// Wrap the private key to implement keychain.SingleKeyECDH interface
	localPriv := &keychain.PrivKeyECDH{PrivKey: privKey}

	// Initialize variables for mailbox connection
	var remotePub *btcec.PublicKey
	var lndConnect func() (*grpc.ClientConn, error)
	var authReceived bool

	// Handle TLS configuration for dev servers - CRITICAL FOR LOCAL CONNECTIONS!
	if devMode || insecure || strings.HasPrefix(mailboxServer, "localhost") ||
		strings.HasPrefix(mailboxServer, "127.0.0.1") {
		zap.L().Info("Configuring insecure connection",
			zap.String("reason", "dev mode or localhost"))
		// This is what the old server did - set global HTTP transport TLS config
		defaultTransport := http.DefaultTransport.(*http.Transport)
		defaultTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		zap.L().Debug("TLS verification disabled for HTTP transport")
	}

	// Create a new mailbox connection
	zap.L().Debug("Creating mailbox WebSocket connection")
	statusChecker, lndConnect, err := mailbox.NewClientWebsocketConn(
		mailboxServer,
		pairingPhrase,
		localPriv,
		remotePub,
		func(key *btcec.PublicKey) error {
			zap.L().Debug("Received remote public key",
				zap.String("key", fmt.Sprintf("%x", key.SerializeCompressed())))
			remotePub = key
			return nil
		},
		func(data []byte) error {
			zap.L().Debug("Received auth data", zap.Int("bytes", len(data)))
			authReceived = true
			return nil
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mailbox connection: %w", err)
	}
	zap.L().Debug("Mailbox connection created successfully")

	// Give some time for the connection callbacks to be triggered (critical!)
	zap.L().Debug("Waiting for connection callbacks to process")
	time.Sleep(3 * time.Second)

	// NEW FIX: Don't wait for status, just check if lndConnect is available
	if lndConnect == nil {
		return nil, nil, fmt.Errorf(
			"lndConnect function not available after connection setup")
	}

	// Wait a bit more for callbacks, but proceed even without them
	maxWaitTime := 5 * time.Second
	waitStart := time.Now()
	zap.L().Debug("Waiting for callbacks (but will proceed anyway)")

	for time.Since(waitStart) < maxWaitTime {
		if authReceived && remotePub != nil {
			zap.L().Debug("All callbacks received")
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	zap.L().Debug("Final connection state",
		zap.Bool("auth_received", authReceived),
		zap.Bool("remote_pub_received", remotePub != nil),
	)
	status := statusChecker()
	zap.L().Debug("Connection status", zap.String("status", status.String()))

	zap.L().Debug("Establishing gRPC connection to LND")
	// Establish gRPC connection to LND
	conn, err := lndConnect()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to establish LND connection: %w", err)
	}
	zap.L().Debug("gRPC connection established successfully")

	// Create lightning client and test connection
	zap.L().Debug("Testing connection with GetInfo")
	lightningClient := lnrpc.NewLightningClient(conn)
	info, err := lightningClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("connected but failed to get node info: %w", err)
	}
	zap.L().Info("Successfully connected to Lightning node",
		zap.String("alias", info.Alias),
		zap.String("pubkey", info.IdentityPubkey),
	)

	return conn, info, nil
}

// DisconnectTool returns the MCP tool definition for disconnecting from LNC.
func (s *ConnectionService) DisconnectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_disconnect",
		Description: "Disconnect from the Lightning node",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}
}

// HandleDisconnect handles the LNC disconnect request.
func (s *ConnectionService) HandleDisconnect(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.Connection != nil {
		s.Connection.Close()
		s.Connection = nil
	}

	return mcp.NewToolResultText(`{
		"disconnected": true,
		"message": "Disconnected from Lightning node"
	}`), nil
}

// getMailboxServer retrieves the mailbox server from tool arguments.
func getMailboxServer(args map[string]any) string {
	if mailbox, ok := args["mailbox"]; ok && mailbox != nil {
		if mailboxStr, ok := mailbox.(string); ok {
			return mailboxStr
		}
	}
	return ""
}
