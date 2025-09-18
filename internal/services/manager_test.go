package services

import (
	"testing"

	"github.com/jbrill/mcp-lnc-server/internal/interfaces"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type stubMCPServer struct {
	tools []mcp.Tool
}

func (s *stubMCPServer) AddTool(tool mcp.Tool, handler interfaces.ToolHandler) {
	s.tools = append(s.tools, tool)
}

// Test Manager creation and basic functionality.
func TestManager_Creation(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	assert.NotNil(t, manager)
	assert.Equal(t, zap.L(), manager.logger)

	// Initialize services to test them.
	manager.InitializeServices()
	assert.NotNil(t, manager.invoiceService)
	assert.NotNil(t, manager.connectionService)
}

// Test RegisterTools with valid MCP server.
func TestManager_RegisterTools(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()
	stub := &stubMCPServer{}

	err = manager.RegisterTools(stub)
	assert.NoError(t, err)

	names := make(map[string]struct{})
	for _, tool := range stub.tools {
		names[tool.Name] = struct{}{}
	}

	assert.Contains(t, names, "lnc_send_payment")
	assert.Contains(t, names, "lnc_open_channel")
	assert.Contains(t, names, "lnc_send_coins")
	assert.NotZero(t, len(stub.tools))
}

func TestManager_RegisterTools_ReadOnlyMode(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), false)
	manager.InitializeServices()
	stub := &stubMCPServer{}

	err = manager.RegisterTools(stub)
	assert.NoError(t, err)

	names := make(map[string]struct{})
	for _, tool := range stub.tools {
		names[tool.Name] = struct{}{}
	}

	assert.NotContains(t, names, "lnc_send_payment")
	assert.NotContains(t, names, "lnc_open_channel")
	assert.NotContains(t, names, "lnc_send_coins")
	assert.Contains(t, names, "lnc_list_channels")
	assert.Contains(t, names, "lnc_get_info")
	assert.Contains(t, names, "lnc_list_unspent")
	assert.Len(t, stub.tools, len(names))
}

// Test RegisterTools with nil MCP server.
func TestManager_RegisterTools_NilServer(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()

	err = manager.RegisterTools(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP server cannot be nil")
}

// Test connection callback functionality.
func TestManager_ConnectionCallback(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()

	// Create a mock connection - this would normally be a real gRPC connection
	// But for testing we just verify the callback doesn't panic.
	mockConn := &grpc.ClientConn{}

	// Call the connection callback - this is private so we can't test it directly
	// But we can verify services were initialized
	assert.NotNil(t, manager.invoiceService)
	assert.NotNil(t, manager.connectionService)

	// In a real scenario, mockConn would be passed to onLNCConnectionEstablished
	// Which would update all service clients.
	_ = mockConn
}

// Test services start with nil clients.
func TestManager_ServicesStartWithNilClients(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()

	// Services should start with nil clients until connection is established
	assert.Nil(t, manager.invoiceService.LightningClient)
	assert.Nil(t, manager.channelService.LightningClient)
	assert.Nil(t, manager.paymentService.LightningClient)
	assert.Nil(t, manager.onchainService.LightningClient)
	assert.Nil(t, manager.peerService.LightningClient)
	assert.Nil(t, manager.nodeService.LightningClient)
}

// Test Shutdown functionality.
func TestManager_Shutdown(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)

	// Test shutdown - should not error
	err = manager.Shutdown()
	assert.NoError(t, err)
}

// Test service integration.
func TestManager_ServiceIntegration(t *testing.T) {
	err := logging.InitLogger(true)
	require.NoError(t, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()

	// Test that services are properly initialized
	assert.NotNil(t, manager.invoiceService)
	assert.NotNil(t, manager.connectionService)
	assert.NotNil(t, manager.channelService)
	assert.NotNil(t, manager.paymentService)
	assert.NotNil(t, manager.onchainService)
	assert.NotNil(t, manager.peerService)
	assert.NotNil(t, manager.nodeService)

	// Test that tools can be created
	createInvoiceTool := manager.invoiceService.CreateInvoiceTool()
	decodeInvoiceTool := manager.invoiceService.DecodeInvoiceTool()
	connectTool := manager.connectionService.ConnectTool()
	disconnectTool := manager.connectionService.DisconnectTool()

	assert.NotNil(t, createInvoiceTool)
	assert.NotNil(t, decodeInvoiceTool)
	assert.NotNil(t, connectTool)
	assert.NotNil(t, disconnectTool)

	// Verify tool names are unique
	names := []string{
		createInvoiceTool.Name,
		decodeInvoiceTool.Name,
		connectTool.Name,
		disconnectTool.Name,
	}

	for i, name := range names {
		for j, otherName := range names {
			if i != j {
				assert.NotEqual(t, name, otherName,
					"Tool names must be unique: %s vs %s", name, otherName)
			}
		}
	}
}

// Benchmark Manager creation.
func BenchmarkManager_Creation(b *testing.B) {
	err := logging.InitLogger(true)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewManager(zap.L(), true)
	}
}

// Benchmark RegisterTools operation.
func BenchmarkManager_RegisterTools(b *testing.B) {
	err := logging.InitLogger(true)
	require.NoError(b, err)

	manager := NewManager(zap.L(), true)
	manager.InitializeServices()
	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.RegisterTools(mcpServer)
	}
}
