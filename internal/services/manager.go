// Package services manages Lightning Network services and their lifecycle.
// It wires MCP tools to underlying clients with consistent logging and error
// handling.
package services

import (
	"context"

	"github.com/jbrill/mcp-lnc-server/internal/errors"
	"github.com/jbrill/mcp-lnc-server/internal/interfaces"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"github.com/jbrill/mcp-lnc-server/tools"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Manager manages all Lightning Network services and their lifecycle.
type Manager struct {
	logger *zap.Logger

	// Global connection and clients.
	lncConnection   *grpc.ClientConn
	lightningClient lnrpc.LightningClient

	// Services - read-only operations only.
	connectionService *tools.ConnectionService
	invoiceService    *tools.InvoiceService
	channelService    *tools.ChannelService
	paymentService    *tools.PaymentService
	onchainService    *tools.OnChainService
	peerService       *tools.PeerService
	nodeService       *tools.NodeService
}

// NewManager creates a new service manager for read-only operations.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// InitializeServices prepares all services with nil clients. Clients are
// provided once an LNC connection is established via the callback.
func (m *Manager) InitializeServices() {
	m.logger.Info("Initializing read-only services...")

	// Initialize connection service with callback.
	m.connectionService = tools.NewConnectionService(
		m.onLNCConnectionEstablished)

	// Initialize all read-only services with nil clients.
	m.invoiceService = tools.NewInvoiceService(nil)
	m.channelService = tools.NewChannelService(nil)
	m.paymentService = tools.NewPaymentService(nil)
	m.onchainService = tools.NewOnChainService(nil)
	m.peerService = tools.NewPeerService(nil)
	m.nodeService = tools.NewNodeService(nil)

	m.logger.Info("Read-only services initialized successfully")
}

// RegisterTools registers all read-only tools with the MCP server.
func (m *Manager) RegisterTools(mcpServer interfaces.MCPServer) error {
	if mcpServer == nil {
		return errors.New(errors.ErrCodeUnknown,
			"MCP server cannot be nil")
	}

	m.logger.Info("Registering read-only MCP tools with server")

	registrations := 0
	register := func(tool mcp.Tool,
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
		mcpServer.AddTool(tool, handler)
		registrations++
	}

	// Connection tools - always required.
	register(m.connectionService.ConnectTool(),
		m.connectionService.HandleConnect)
	register(m.connectionService.DisconnectTool(),
		m.connectionService.HandleDisconnect)

	// Invoice tools - read-only operations.
	register(m.invoiceService.DecodeInvoiceTool(),
		m.invoiceService.HandleDecodeInvoice)
	register(m.invoiceService.ListInvoicesTool(),
		m.invoiceService.HandleListInvoices)
	register(m.invoiceService.LookupInvoiceTool(),
		m.invoiceService.HandleLookupInvoice)

	// Channel tools - read-only operations.
	register(m.channelService.ListChannelsTool(),
		m.channelService.HandleListChannels)
	register(m.channelService.PendingChannelsTool(),
		m.channelService.HandlePendingChannels)

	// Payment tools - read-only operations.
	register(m.paymentService.ListPaymentsTool(),
		m.paymentService.HandleListPayments)
	register(m.paymentService.TrackPaymentTool(),
		m.paymentService.HandleTrackPayment)

	// On-chain tools - read-only operations.
	register(m.onchainService.ListUnspentTool(),
		m.onchainService.HandleListUnspent)
	register(m.onchainService.GetTransactionsTool(),
		m.onchainService.HandleGetTransactions)
	register(m.onchainService.EstimateFeesTool(),
		m.onchainService.HandleEstimateFee)

	// Peer tools - read-only operations.
	register(m.peerService.ListPeersTool(),
		m.peerService.HandleListPeers)
	register(m.peerService.DescribeGraphTool(),
		m.peerService.HandleDescribeGraph)
	register(m.peerService.GetNodeInfoTool(),
		m.peerService.HandleGetNodeInfo)

	// Node tools - read-only operations.
	register(m.nodeService.GetBalanceTool(),
		m.nodeService.HandleGetBalance)
	register(m.nodeService.GetInfoTool(),
		m.nodeService.HandleGetInfo)

	m.logger.Info("Read-only MCP tools registered",
		zap.Int("total_tools", registrations))
	return nil
}

// onLNCConnectionEstablished updates service clients when a new LNC
// connection becomes available.
func (m *Manager) onLNCConnectionEstablished(conn *grpc.ClientConn) {
	logger := logging.LogWithContext(context.Background())
	logger.Info("LNC connection established successfully")

	m.lncConnection = conn
	m.lightningClient = lnrpc.NewLightningClient(conn)

	// Update existing read-only services with new connection.
	m.invoiceService.LightningClient = m.lightningClient
	m.channelService.LightningClient = m.lightningClient
	m.paymentService.LightningClient = m.lightningClient
	m.onchainService.LightningClient = m.lightningClient
	m.peerService.LightningClient = m.lightningClient
	m.nodeService.LightningClient = m.lightningClient

	logger.Info("All read-only services updated with new connection")
}

// Shutdown gracefully closes the LNC connection and logs shutdown results.
func (m *Manager) Shutdown() error {
	m.logger.Info("Shutting down service manager...")

	if m.lncConnection != nil {
		if err := m.lncConnection.Close(); err != nil {
			m.logger.Error("Error closing LNC connection",
				zap.Error(err))
			return errors.Wrap(err, errors.ErrCodeUnknown,
				"failed to close LNC connection")
		} else {
			m.logger.Info("LNC connection closed successfully")
		}
	}

	m.logger.Info("Service manager shutdown complete")
	return nil
}
