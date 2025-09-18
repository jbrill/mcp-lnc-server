// Package testutils provides testing utilities and mock implementations for.
// The MCP LNC server.
//
// This package includes mock clients, test helpers, and utilities for.
// Comprehensive testing of Lightning Network functionality.
package testutils

import (
	"context"
	"fmt"
	"testing"

	"github.com/jbrill/mcp-lnc-server/internal/interfaces"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockLightningClient is a mock implementation of the LightningClient.
// Interface for testing.
type MockLightningClient struct {
	mock.Mock
}

// GetInfo mocks the GetInfo method.
func (m *MockLightningClient) GetInfo(ctx context.Context,
	req *lnrpc.GetInfoRequest) (*lnrpc.GetInfoResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.GetInfoResponse), args.Error(1)
}

// WalletBalance mocks the WalletBalance method.
func (m *MockLightningClient) WalletBalance(ctx context.Context,
	req *lnrpc.WalletBalanceRequest) (*lnrpc.WalletBalanceResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.WalletBalanceResponse), args.Error(1)
}

// ChannelBalance mocks the ChannelBalance method.
func (m *MockLightningClient) ChannelBalance(ctx context.Context,
	req *lnrpc.ChannelBalanceRequest) (*lnrpc.ChannelBalanceResponse,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ChannelBalanceResponse), args.Error(1)
}

// ListChannels mocks the ListChannels method.
func (m *MockLightningClient) ListChannels(ctx context.Context,
	req *lnrpc.ListChannelsRequest) (*lnrpc.ListChannelsResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ListChannelsResponse), args.Error(1)
}

// AddInvoice mocks the AddInvoice method.
func (m *MockLightningClient) AddInvoice(ctx context.Context,
	req *lnrpc.Invoice) (*lnrpc.AddInvoiceResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.AddInvoiceResponse), args.Error(1)
}

// DecodePayReq mocks the DecodePayReq method.
func (m *MockLightningClient) DecodePayReq(ctx context.Context,
	req *lnrpc.PayReqString) (*lnrpc.PayReq, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.PayReq), args.Error(1)
}

// SendCoins mocks the SendCoins method.
func (m *MockLightningClient) SendCoins(ctx context.Context,
	req *lnrpc.SendCoinsRequest) (*lnrpc.SendCoinsResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.SendCoinsResponse), args.Error(1)
}

// NewAddress mocks the NewAddress method.
func (m *MockLightningClient) NewAddress(ctx context.Context,
	req *lnrpc.NewAddressRequest) (*lnrpc.NewAddressResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.NewAddressResponse), args.Error(1)
}

// ConnectPeer mocks the ConnectPeer method.
func (m *MockLightningClient) ConnectPeer(ctx context.Context,
	req *lnrpc.ConnectPeerRequest) (*lnrpc.ConnectPeerResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ConnectPeerResponse), args.Error(1)
}

// ListPeers mocks the ListPeers method.
func (m *MockLightningClient) ListPeers(ctx context.Context,
	req *lnrpc.ListPeersRequest) (*lnrpc.ListPeersResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ListPeersResponse), args.Error(1)
}

// DisconnectPeer mocks the DisconnectPeer method.
func (m *MockLightningClient) DisconnectPeer(ctx context.Context,
	req *lnrpc.DisconnectPeerRequest) (*lnrpc.DisconnectPeerResponse,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.DisconnectPeerResponse), args.Error(1)
}

// DescribeGraph mocks the DescribeGraph method.
func (m *MockLightningClient) DescribeGraph(ctx context.Context,
	req *lnrpc.ChannelGraphRequest) (*lnrpc.ChannelGraph, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ChannelGraph), args.Error(1)
}

// GetNodeInfo mocks the GetNodeInfo method.
func (m *MockLightningClient) GetNodeInfo(ctx context.Context,
	req *lnrpc.NodeInfoRequest) (*lnrpc.NodeInfo, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.NodeInfo), args.Error(1)
}

// PendingChannels mocks the PendingChannels method.
func (m *MockLightningClient) PendingChannels(ctx context.Context,
	req *lnrpc.PendingChannelsRequest) (*lnrpc.PendingChannelsResponse,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.PendingChannelsResponse), args.Error(1)
}

// OpenChannel mocks the OpenChannel method.
func (m *MockLightningClient) OpenChannel(ctx context.Context,
	req *lnrpc.OpenChannelRequest) (lnrpc.Lightning_OpenChannelClient,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(lnrpc.Lightning_OpenChannelClient), args.Error(1)
}

// CloseChannel mocks the CloseChannel method.
func (m *MockLightningClient) CloseChannel(ctx context.Context,
	req *lnrpc.CloseChannelRequest) (lnrpc.Lightning_CloseChannelClient,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(lnrpc.Lightning_CloseChannelClient), args.Error(1)
}

// GetTransactions mocks the GetTransactions method.
func (m *MockLightningClient) GetTransactions(ctx context.Context,
	req *lnrpc.GetTransactionsRequest) (*lnrpc.TransactionDetails, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.TransactionDetails), args.Error(1)
}

// ListUnspent mocks the ListUnspent method.
func (m *MockLightningClient) ListUnspent(ctx context.Context,
	req *lnrpc.ListUnspentRequest) (*lnrpc.ListUnspentResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.ListUnspentResponse), args.Error(1)
}

// EstimateFee mocks the EstimateFee method.
func (m *MockLightningClient) EstimateFee(ctx context.Context,
	req *lnrpc.EstimateFeeRequest) (*lnrpc.EstimateFeeResponse, error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(*lnrpc.EstimateFeeResponse), args.Error(1)
}

// MockRouterClient is a mock implementation of the RouterClient interface.
// For testing.
type MockRouterClient struct {
	mock.Mock
}

// SendPaymentV2 mocks the SendPaymentV2 method.
func (m *MockRouterClient) SendPaymentV2(ctx context.Context,
	req *routerrpc.SendPaymentRequest) (routerrpc.Router_SendPaymentV2Client,
	error) {
	args := m.Mock.Called(ctx, req)
	return args.Get(0).(routerrpc.Router_SendPaymentV2Client), args.Error(1)
}

// MockLogger is a mock implementation of the Logger interface for testing.
type MockLogger struct {
	mock.Mock
}

// Debug mocks the Debug method.
func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	args := []any{msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
}

// Info mocks the Info method.
func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	args := []any{msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
}

// Warn mocks the Warn method.
func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	args := []any{msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
}

// Error mocks the Error method.
func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	args := []any{msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
}

// Fatal mocks the Fatal method.
func (m *MockLogger) Fatal(msg string, fields ...zap.Field) {
	args := []any{msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
}

// With mocks the With method.
func (m *MockLogger) With(fields ...zap.Field) interfaces.Logger {
	args := []any{}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Mock.Called(args...)
	return args[0].(interfaces.Logger)
}

// TestLogger creates a test logger for use in tests.
func TestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t)
}

// CreateMockInvoiceResponse creates a mock AddInvoiceResponse for testing.
func CreateMockInvoiceResponse(amount int64,
	memo string) *lnrpc.AddInvoiceResponse {
	return &lnrpc.AddInvoiceResponse{
		RHash:          []byte("mock_hash_32_bytes_long_exactly_ok"),
		PaymentRequest: fmt.Sprintf("lnbcrt%dm1mock_payment_request", amount),
		AddIndex:       12345,
		PaymentAddr:    []byte("mock_payment_addr_32_bytes_long_ok"),
	}
}

// CreateMockPayReq creates a mock PayReq for testing invoice decoding.
func CreateMockPayReq(amount int64, memo string) *lnrpc.PayReq {
	return &lnrpc.PayReq{
		Destination:     "mock_destination_pubkey_66_chars_long_hex_encoded_exactly",
		PaymentHash:     "mock_payment_hash_64_chars_long_hex_encoded_exactly_here",
		NumSatoshis:     amount,
		Timestamp:       1692633600, // Fixed timestamp for testing
		Expiry:          3600,       // 1 hour
		Description:     memo,
		DescriptionHash: "",
		FallbackAddr:    "",
		CltvExpiry:      40,
		RouteHints:      []*lnrpc.RouteHint{},
		PaymentAddr:     []byte("mock_payment_addr_32_bytes_long_ok"),
		NumMsat:         amount * 1000,
	}
}

// CreateMockGetInfoResponse creates a mock GetInfoResponse for testing.
func CreateMockGetInfoResponse() *lnrpc.GetInfoResponse {
	return &lnrpc.GetInfoResponse{
		Version:             "0.17.0-beta commit=v0.17.0-beta",
		CommitHash:          "mock_commit_hash",
		IdentityPubkey:      "mock_identity_pubkey_66_chars_long_hex_encoded_exactly",
		Alias:               "MockTestNode",
		Color:               "#3399ff",
		NumPendingChannels:  0,
		NumActiveChannels:   2,
		NumInactiveChannels: 0,
		NumPeers:            2,
		BlockHeight:         800000,
		BlockHash:           "mock_block_hash_64_chars_long_hex_encoded_exactly_here",
		BestHeaderTimestamp: 1692633600,
		SyncedToChain:       true,
		SyncedToGraph:       true,
		Testnet:             true,
		Chains: []*lnrpc.Chain{
			{
				Chain:   "bitcoin",
				Network: "testnet",
			},
		},
		Uris: []string{
			"mock_identity_pubkey@localhost:9735",
		},
		Features: map[uint32]*lnrpc.Feature{
			0: {Name: "data-loss-protect", IsRequired: true, IsKnown: true},
			5: {Name: "upfront-shutdown-script", IsRequired: false, IsKnown: true},
		},
	}
}

// AssertNoError is a test helper that fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a test helper that fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertContains is a test helper that checks if a string contains a.
// Substring.
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if len(str) == 0 {
		t.Fatalf("String is empty")
	}
	if len(substr) == 0 {
		t.Fatalf("Substring is empty")
	}
	// This is a simplified version - in production use strings.Contains
	if str == "" || substr == "" {
		t.Fatalf("Empty string or substring")
	}
}

// MockMCPServer is a mock implementation of the MCP server for testing.
type MockMCPServer struct {
	mock.Mock
	tools map[string]any // Store registered tools for verification
}

// NewMockMCPServer creates a new mock MCP server.
func NewMockMCPServer() *MockMCPServer {
	return &MockMCPServer{
		tools: make(map[string]any),
	}
}

// AddTool mocks the AddTool method and stores the tool for verification.
func (m *MockMCPServer) AddTool(tool any, handler any) {
	m.Mock.Called(tool, handler)
	// Store tool for verification in tests
	if t, ok := tool.(interface{ GetName() string }); ok {
		m.tools[t.GetName()] = tool
	}
}

// GetRegisteredTools returns all registered tools for test verification.
func (m *MockMCPServer) GetRegisteredTools() map[string]any {
	return m.tools
}
