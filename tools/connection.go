// Package tools provides MCP tool implementations for Lightning Network operations.
// It enables AI assistants to interact with nodes through dedicated services.
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
	lnccontext "github.com/jbrill/mcp-lnc-server/internal/context"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
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
	// Create request context with tracing
	reqCtx := lnccontext.New(ctx, "lnc_connect", 45*time.Second)
	defer reqCtx.Cancel()
	logger := logging.LogWithContext(reqCtx)

	logger.Info("Starting LNC connection request",
		zap.Any("params", request.Params.Arguments))

	defer func() {
		logger.Info("Connection request completed",
			zap.Duration("total_duration", reqCtx.Duration()))
	}()

	pairingPhrase, ok := request.Params.Arguments["pairingPhrase"].(string)
	if !ok {
		logger.Error("Missing pairing phrase in request")
		return mcp.NewToolResultError("pairingPhrase is required"), nil
	}

	password, ok := request.Params.Arguments["password"].(string)
	if !ok {
		logger.Error("Missing password in request")
		return mcp.NewToolResultError("password is required"), nil
	}

	// Validate pairing phrase format
	words := strings.Split(strings.TrimSpace(pairingPhrase), " ")
	if len(words) != 10 {
		logger.Error("Invalid pairing phrase format",
			zap.Int("word_count", len(words)))
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

	// Use request context for connection (it already has timeout)
	logger.Info("Attempting LNC connection",
		zap.String("mailbox", mailboxServer),
		zap.Bool("devMode", devMode),
		zap.Bool("insecure", insecure),
		zap.Duration("timeout", timeout),
	)

	// Establish LNC connection
	conn, nodeInfo, err := s.connectToLNC(reqCtx, pairingPhrase,
		password, mailboxServer, devMode, insecure)
	if err != nil {
		logger.Error("LNC connection failed",
			zap.Error(err),
			zap.Duration("failed_after", reqCtx.Duration()))
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to connect to Lightning node: %v", err)), nil
	}

	// Store connection
	s.Connection = conn

	// Add node ID to context for future operations
	reqCtx = reqCtx.WithNode(nodeInfo.IdentityPubkey)

	// Notify main server of new connection
	if s.ConnectionCallback != nil {
		s.ConnectionCallback(conn)
	}

	logger.Info("Successfully connected to Lightning node",
		zap.String("node_pubkey", nodeInfo.IdentityPubkey),
		zap.String("alias", nodeInfo.Alias),
		zap.Uint32("num_channels", nodeInfo.NumActiveChannels),
		zap.Uint32("num_peers", nodeInfo.NumPeers))

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

// ConnectToLNC establishes the actual LNC connection.
func (s *ConnectionService) connectToLNC(ctx context.Context,
	pairingPhrase, password, mailboxServer string, devMode,
	insecure bool) (*grpc.ClientConn, *lnrpc.GetInfoResponse, error) {

	// Ensure we have a RequestContext
	reqCtx := lnccontext.Ensure(ctx, "lnc_connect_internal")
	defer reqCtx.Cancel()
	logger := logging.LogWithContext(reqCtx)

	logger.Debug("Starting LNC connection process",
		zap.String("mailbox", mailboxServer),
		zap.Int("pairing_phrase_words", len(strings.Split(pairingPhrase, " "))),
		zap.Bool("dev_mode", devMode),
		zap.Bool("insecure", insecure),
		zap.Bool("has_password", password != ""),
	)

	// Generate a new private key for this session
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		logger.Error("Failed to generate private key", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	logger.Debug("Generated session private key")

	// Wrap the private key to implement keychain.SingleKeyECDH interface
	localPriv := &keychain.PrivKeyECDH{PrivKey: privKey}

	// Initialize variables for mailbox connection
	var remotePub *btcec.PublicKey
	var lndConnect func() (*grpc.ClientConn, error)
	var authReceived bool

	// Handle TLS configuration for dev servers - CRITICAL FOR LOCAL CONNECTIONS!
	if devMode || insecure || strings.HasPrefix(mailboxServer, "localhost") ||
		strings.HasPrefix(mailboxServer, "127.0.0.1") {
		logger.Info("Configuring insecure connection",
			zap.String("reason", "dev mode or localhost"))
		// This is what the old server did - set global HTTP transport TLS config
		defaultTransport := http.DefaultTransport.(*http.Transport)
		defaultTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		logger.Debug("TLS verification disabled for HTTP transport")
	}

	// Create a new mailbox connection
	logger.Debug("Creating mailbox WebSocket connection")
	statusChecker, lndConnect, err := mailbox.NewClientWebsocketConn(
		mailboxServer,
		pairingPhrase,
		localPriv,
		remotePub,
		func(key *btcec.PublicKey) error {
			logger.Debug("Received remote public key",
				zap.String("key", fmt.Sprintf("%x", key.SerializeCompressed())))
			remotePub = key
			return nil
		},
		func(data []byte) error {
			logger.Debug("Received auth data", zap.Int("bytes", len(data)))
			authReceived = true
			return nil
		},
	)
	if err != nil {
		logger.Error("Failed to create mailbox connection",
			zap.Error(err),
			zap.Duration("failed_after", reqCtx.Duration()))
		return nil, nil, fmt.Errorf("failed to create mailbox connection: %w", err)
	}
	logger.Debug("Mailbox connection created successfully")

	// Give some time for the connection callbacks to be triggered (critical!)
	logger.Debug("Waiting for connection callbacks to process")

	// Check for context cancellation during wait
	select {
	case <-time.After(3 * time.Second):
		// Continue
	case <-reqCtx.Done():
		logger.Error("Context cancelled during callback wait")
		return nil, nil, fmt.Errorf("connection cancelled: %w", reqCtx.Err())
	}

	// NEW FIX: Don't wait for status, just check if lndConnect is available
	if lndConnect == nil {
		logger.Error("lndConnect function not available after connection setup")
		return nil, nil, fmt.Errorf(
			"lndConnect function not available after connection setup")
	}

	// Wait a bit more for callbacks, but proceed even without them
	maxWaitTime := 5 * time.Second
	waitStart := time.Now()
	logger.Debug("Waiting for callbacks (but will proceed anyway)")

	for time.Since(waitStart) < maxWaitTime {
		// Check for context cancellation
		select {
		case <-reqCtx.Done():
			logger.Error("Context cancelled during auth wait")
			return nil, nil, fmt.Errorf("connection cancelled: %w", reqCtx.Err())
		default:
		}

		if authReceived && remotePub != nil {
			logger.Debug("All callbacks received")
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	logger.Debug("Final connection state",
		zap.Bool("auth_received", authReceived),
		zap.Bool("remote_pub_received", remotePub != nil),
		zap.Duration("elapsed", reqCtx.Duration()),
	)
	status := statusChecker()
	logger.Debug("Connection status", zap.String("status", status.String()))

	logger.Debug("Establishing gRPC connection to LND")
	// Establish gRPC connection to LND
	conn, err := lndConnect()
	if err != nil {
		logger.Error("Failed to establish LND connection",
			zap.Error(err),
			zap.Duration("failed_after", reqCtx.Duration()))
		return nil, nil, fmt.Errorf("failed to establish LND connection: %w", err)
	}
	logger.Debug("gRPC connection established successfully")

	// Create lightning client and test connection
	logger.Debug("Testing connection with GetInfo")
	lightningClient := lnrpc.NewLightningClient(conn)
	info, err := lightningClient.GetInfo(reqCtx, &lnrpc.GetInfoRequest{})
	if err != nil {
		logger.Error("Failed to get node info",
			zap.Error(err),
			zap.Duration("failed_after", reqCtx.Duration()))
		conn.Close()
		return nil, nil, fmt.Errorf("connected but failed to get node info: %w", err)
	}
	logger.Info("Successfully connected to Lightning node",
		zap.String("alias", info.Alias),
		zap.String("pubkey", info.IdentityPubkey),
		zap.Duration("total_connection_time", reqCtx.Duration()),
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
	// Create request context
	reqCtx := lnccontext.New(ctx, "lnc_disconnect", 10*time.Second)
	defer reqCtx.Cancel()
	logger := logging.LogWithContext(reqCtx)

	logger.Info("Disconnecting from Lightning node")

	if s.Connection != nil {
		err := s.Connection.Close()
		if err != nil {
			logger.Error("Error closing connection", zap.Error(err))
		} else {
			logger.Info("Connection closed successfully")
		}
		s.Connection = nil
	} else {
		logger.Debug("No active connection to close")
	}

	return mcp.NewToolResultText(`{
		"disconnected": true,
		"message": "Disconnected from Lightning node"
	}`), nil
}

// GetMailboxServer retrieves the mailbox server from tool arguments.
func getMailboxServer(args map[string]any) string {
	if mailbox, ok := args["mailbox"]; ok && mailbox != nil {
		if mailboxStr, ok := mailbox.(string); ok {
			return mailboxStr
		}
	}
	return ""
}
