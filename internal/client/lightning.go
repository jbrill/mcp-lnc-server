// Package client provides Lightning Network client wrappers that implement.
// Our defined interfaces.
//
// This allows us to follow the "accept interfaces, return structs".
// Principle while maintaining compatibility with the LND gRPC clients.
package client

import (
	"context"

	"github.com/jbrill/mcp-lnc-server/internal/interfaces"
	"github.com/lightningnetwork/lnd/lnrpc"
)

// LightningClientWrapper wraps the LND Lightning client to implement.
// Our LightningClient interface.
type lightningClientWrapper struct {
	client lnrpc.LightningClient
}

// NewLightningClient creates a new Lightning client wrapper.
func NewLightningClient(
	client lnrpc.LightningClient) interfaces.LightningClient {
	return &lightningClientWrapper{client: client}
}

// GetInfo retrieves general information about the Lightning node.
func (w *lightningClientWrapper) GetInfo(ctx context.Context,
	req *lnrpc.GetInfoRequest) (*lnrpc.GetInfoResponse, error) {
	return w.client.GetInfo(ctx, req)
}

// WalletBalance retrieves the on-chain wallet balance.
func (w *lightningClientWrapper) WalletBalance(ctx context.Context,
	req *lnrpc.WalletBalanceRequest) (
	*lnrpc.WalletBalanceResponse, error) {
	return w.client.WalletBalance(ctx, req)
}

// ChannelBalance retrieves the Lightning channel balance.
func (w *lightningClientWrapper) ChannelBalance(ctx context.Context,
	req *lnrpc.ChannelBalanceRequest) (
	*lnrpc.ChannelBalanceResponse, error) {
	return w.client.ChannelBalance(ctx, req)
}

// ListChannels lists all Lightning channels.
func (w *lightningClientWrapper) ListChannels(ctx context.Context,
	req *lnrpc.ListChannelsRequest) (
	*lnrpc.ListChannelsResponse, error) {
	return w.client.ListChannels(ctx, req)
}

// AddInvoice creates a new Lightning invoice.
func (w *lightningClientWrapper) AddInvoice(ctx context.Context,
	req *lnrpc.Invoice) (*lnrpc.AddInvoiceResponse, error) {
	return w.client.AddInvoice(ctx, req)
}

// DecodePayReq decodes a BOLT11 payment request.
func (w *lightningClientWrapper) DecodePayReq(ctx context.Context,
	req *lnrpc.PayReqString) (*lnrpc.PayReq, error) {
	return w.client.DecodePayReq(ctx, req)
}

// SendCoins sends an on-chain transaction.
func (w *lightningClientWrapper) SendCoins(ctx context.Context,
	req *lnrpc.SendCoinsRequest) (*lnrpc.SendCoinsResponse, error) {
	return w.client.SendCoins(ctx, req)
}

// NewAddress generates a new on-chain address.
func (w *lightningClientWrapper) NewAddress(ctx context.Context,
	req *lnrpc.NewAddressRequest) (
	*lnrpc.NewAddressResponse, error) {
	return w.client.NewAddress(ctx, req)
}

// ConnectPeer connects to a Lightning Network peer.
func (w *lightningClientWrapper) ConnectPeer(ctx context.Context,
	req *lnrpc.ConnectPeerRequest) (
	*lnrpc.ConnectPeerResponse, error) {
	return w.client.ConnectPeer(ctx, req)
}

// ListPeers lists all connected Lightning Network peers.
func (w *lightningClientWrapper) ListPeers(ctx context.Context,
	req *lnrpc.ListPeersRequest) (*lnrpc.ListPeersResponse, error) {
	return w.client.ListPeers(ctx, req)
}

// DisconnectPeer disconnects from a Lightning Network peer.
func (w *lightningClientWrapper) DisconnectPeer(ctx context.Context,
	req *lnrpc.DisconnectPeerRequest) (
	*lnrpc.DisconnectPeerResponse, error) {
	return w.client.DisconnectPeer(ctx, req)
}

// DescribeGraph retrieves the Lightning Network graph.
func (w *lightningClientWrapper) DescribeGraph(ctx context.Context,
	req *lnrpc.ChannelGraphRequest) (*lnrpc.ChannelGraph, error) {
	return w.client.DescribeGraph(ctx, req)
}

// GetNodeInfo retrieves information about a specific node.
func (w *lightningClientWrapper) GetNodeInfo(ctx context.Context,
	req *lnrpc.NodeInfoRequest) (*lnrpc.NodeInfo, error) {
	return w.client.GetNodeInfo(ctx, req)
}

// PendingChannels lists all pending Lightning channels.
func (w *lightningClientWrapper) PendingChannels(ctx context.Context,
	req *lnrpc.PendingChannelsRequest) (
	*lnrpc.PendingChannelsResponse, error) {
	return w.client.PendingChannels(ctx, req)
}

// OpenChannel opens a new Lightning channel.
func (w *lightningClientWrapper) OpenChannel(ctx context.Context,
	req *lnrpc.OpenChannelRequest) (
	lnrpc.Lightning_OpenChannelClient, error) {
	return w.client.OpenChannel(ctx, req)
}

// CloseChannel closes an existing Lightning channel.
func (w *lightningClientWrapper) CloseChannel(ctx context.Context,
	req *lnrpc.CloseChannelRequest) (
	lnrpc.Lightning_CloseChannelClient, error) {
	return w.client.CloseChannel(ctx, req)
}

// GetTransactions retrieves on-chain transaction history.
func (w *lightningClientWrapper) GetTransactions(ctx context.Context,
	req *lnrpc.GetTransactionsRequest) (
	*lnrpc.TransactionDetails, error) {
	return w.client.GetTransactions(ctx, req)
}

// ListUnspent lists unspent transaction outputs.
func (w *lightningClientWrapper) ListUnspent(ctx context.Context,
	req *lnrpc.ListUnspentRequest) (
	*lnrpc.ListUnspentResponse, error) {
	return w.client.ListUnspent(ctx, req)
}

// EstimateFee estimates on-chain transaction fees.
func (w *lightningClientWrapper) EstimateFee(ctx context.Context,
	req *lnrpc.EstimateFeeRequest) (
	*lnrpc.EstimateFeeResponse, error) {
	return w.client.EstimateFee(ctx, req)
}
