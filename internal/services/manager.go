// Package services manages all Lightning Network services and their lifecycle.
//
// Following LND contribution guidelines, this package provides clean
// service management with proper error handling, structured logging,
// and adherence to Go best practices.
package services

import (
	"github.com/jbrill/mcp-lnc-server/internal/errors"
	"github.com/jbrill/mcp-lnc-server/tools"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Manager manages all Lightning Network services and their lifecycle
type Manager struct {
	logger *zap.Logger

	// Global connection and clients
	lncConnection   *grpc.ClientConn
	lightningClient lnrpc.LightningClient
	invoicesClient  invoicesrpc.InvoicesClient
	routerClient    routerrpc.RouterClient

	// Services
	connectionService *tools.ConnectionService
	invoiceService    *tools.InvoiceService
	channelService    *tools.ChannelService
	paymentService    *tools.PaymentService
	onchainService    *tools.OnChainService
	peerService       *tools.PeerService
	nodeService       *tools.NodeService
}

// NewManager creates a new service manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// InitializeServices initializes all services with nil clients.
//
// Services start with nil clients and are updated when connection is
// established through the onLNCConnectionEstablished callback.
func (m *Manager) InitializeServices() {
	m.logger.Info("Initializing services...")

	// Initialize connection service with callback
	m.connectionService = tools.NewConnectionService(
		m.onLNCConnectionEstablished)

	// Initialize all other services with nil clients (they'll check
	// for connection)
	m.invoiceService = tools.NewInvoiceService(nil)
	m.channelService = tools.NewChannelService(nil)
	m.paymentService = tools.NewPaymentService(nil, nil)
	m.onchainService = tools.NewOnChainService(nil)
	m.peerService = tools.NewPeerService(nil)
	m.nodeService = tools.NewNodeService(nil)

	m.logger.Info("Services initialized successfully")
}

// RegisterTools registers all tools with the MCP server.
//
// This method follows LND patterns by using structured logging and
// providing detailed tool registration information.
func (m *Manager) RegisterTools(mcpServer *server.MCPServer) error {
	if mcpServer == nil {
		return errors.New(errors.ErrCodeUnknown,
			"MCP server cannot be nil")
	}

	m.logger.Info("Registering MCP tools with server")

	// Connection tools - always required
	mcpServer.AddTool(m.connectionService.ConnectTool(),
		m.connectionService.HandleConnect)
	mcpServer.AddTool(m.connectionService.DisconnectTool(),
		m.connectionService.HandleDisconnect)

	// Invoice tools
	mcpServer.AddTool(m.invoiceService.CreateInvoiceTool(),
		m.invoiceService.HandleCreateInvoice)
	mcpServer.AddTool(m.invoiceService.DecodeInvoiceTool(),
		m.invoiceService.HandleDecodeInvoice)

	// Channel tools
	mcpServer.AddTool(m.channelService.ListChannelsTool(),
		m.channelService.HandleListChannels)
	mcpServer.AddTool(m.channelService.PendingChannelsTool(),
		m.channelService.HandlePendingChannels)
	mcpServer.AddTool(m.channelService.OpenChannelTool(),
		m.channelService.HandleOpenChannel)
	mcpServer.AddTool(m.channelService.CloseChannelTool(),
		m.channelService.HandleCloseChannel)

	// Payment tools
	mcpServer.AddTool(m.paymentService.SendPaymentTool(),
		m.paymentService.HandleSendPayment)
	mcpServer.AddTool(m.paymentService.PayInvoiceTool(),
		m.paymentService.HandlePayInvoice)

	// On-chain tools
	mcpServer.AddTool(m.onchainService.SendCoinsTool(),
		m.onchainService.HandleSendCoins)
	mcpServer.AddTool(m.onchainService.NewAddressTool(),
		m.onchainService.HandleNewAddress)
	mcpServer.AddTool(m.onchainService.ListUnspentTool(),
		m.onchainService.HandleListUnspent)
	mcpServer.AddTool(m.onchainService.GetTransactionsTool(),
		m.onchainService.HandleGetTransactions)
	mcpServer.AddTool(m.onchainService.EstimateFeesTool(),
		m.onchainService.HandleEstimateFee)

	// Peer tools
	mcpServer.AddTool(m.peerService.ListPeersTool(),
		m.peerService.HandleListPeers)
	mcpServer.AddTool(m.peerService.ConnectPeerTool(),
		m.peerService.HandleConnectPeer)
	mcpServer.AddTool(m.peerService.DisconnectPeerTool(),
		m.peerService.HandleDisconnectPeer)
	mcpServer.AddTool(m.peerService.DescribeGraphTool(),
		m.peerService.HandleDescribeGraph)
	mcpServer.AddTool(m.peerService.GetNodeInfoTool(),
		m.peerService.HandleGetNodeInfo)

	// Node tools
	mcpServer.AddTool(m.nodeService.GetBalanceTool(),
		m.nodeService.HandleGetBalance)
	mcpServer.AddTool(m.nodeService.GetInfoTool(),
		m.nodeService.HandleGetInfo)

	m.logger.Info("All MCP tools registered successfully",
		zap.Int("total_tools", 21))
	return nil
}

// onLNCConnectionEstablished is called when LNC connection is
// established
func (m *Manager) onLNCConnectionEstablished(conn *grpc.ClientConn) {
	m.logger.Info("LNC connection established successfully")

	m.lncConnection = conn
	m.lightningClient = lnrpc.NewLightningClient(conn)
	m.invoicesClient = invoicesrpc.NewInvoicesClient(conn)
	m.routerClient = routerrpc.NewRouterClient(conn)

	// Update existing services with new connection (they're already
	// registered)
	m.invoiceService.LightningClient = m.lightningClient
	m.channelService.LightningClient = m.lightningClient
	m.paymentService.LightningClient = m.lightningClient
	m.paymentService.RouterClient = m.routerClient
	m.onchainService.LightningClient = m.lightningClient
	m.peerService.LightningClient = m.lightningClient
	m.nodeService.LightningClient = m.lightningClient

	m.logger.Info(
		"All Lightning Network services updated with new connection")
}

// Shutdown gracefully shuts down all services and closes connections.
//
// This method ensures proper cleanup of all resources, following
// LND patterns for graceful shutdown with error logging.
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
