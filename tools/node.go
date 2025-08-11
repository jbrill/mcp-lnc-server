package tools

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// NodeService handles Lightning node information operations.
type NodeService struct {
	LightningClient lnrpc.LightningClient
}

// NewNodeService creates a new node service.
func NewNodeService(client lnrpc.LightningClient) *NodeService {
	return &NodeService{
		LightningClient: client,
	}
}

// GetInfoTool returns the MCP tool definition for getting node info.
func (s *NodeService) GetInfoTool() mcp.Tool {
	return mcp.Tool{
		Name: "lnc_get_info",
		Description: "Get Lightning node information including version, " +
			"peers, and channels",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}
}

// HandleGetInfo handles the node info request.
func (s *NodeService) HandleGetInfo(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	info, err := s.LightningClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to get node info: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"node_id": "%s",
		"alias": "%s",
		"version": "%s",
		"num_peers": %d,
		"num_active_channels": %d,
		"num_inactive_channels": %d,
		"num_pending_channels": %d,
		"synced_to_chain": %t,
		"synced_to_graph": %t,
		"block_height": %d,
		"block_hash": "%s",
		"testnet": %t,
		"chains": %v
	}`,
		info.IdentityPubkey,
		info.Alias,
		info.Version,
		info.NumPeers,
		info.NumActiveChannels,
		info.NumInactiveChannels,
		info.NumPendingChannels,
		info.SyncedToChain,
		info.SyncedToGraph,
		info.BlockHeight,
		info.BlockHash,
		info.Testnet,
		chainNames(info.Chains),
	)), nil
}

// GetBalanceTool returns the MCP tool definition for getting wallet balance.
func (s *NodeService) GetBalanceTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_get_balance",
		Description: "Get on-chain wallet balance and channel balance information",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}
}

// HandleGetBalance handles the balance request.
func (s *NodeService) HandleGetBalance(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	// Get on-chain balance
	walletBalance, err := s.LightningClient.WalletBalance(ctx,
		&lnrpc.WalletBalanceRequest{})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to get wallet balance: %v", err)), nil
	}

	// Get channel balance
	channelBalance, err := s.LightningClient.ChannelBalance(ctx,
		&lnrpc.ChannelBalanceRequest{})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to get channel balance: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"wallet_balance": {
			"total_balance": %d,
			"confirmed_balance": %d,
			"unconfirmed_balance": %d
		},
		"channel_balance": {
			"balance": %d,
			"pending_open_balance": %d,
			"local_balance": {
				"sat": %d,
				"msat": %d
			},
			"remote_balance": {
				"sat": %d,
				"msat": %d
			},
			"unsettled_local_balance": {
				"sat": %d,
				"msat": %d
			},
			"unsettled_remote_balance": {
				"sat": %d,
				"msat": %d
			}
		}
	}`,
		walletBalance.TotalBalance,
		walletBalance.ConfirmedBalance,
		walletBalance.UnconfirmedBalance,
		channelBalance.Balance,
		channelBalance.PendingOpenBalance,
		channelBalance.LocalBalance.Sat,
		channelBalance.LocalBalance.Msat,
		channelBalance.RemoteBalance.Sat,
		channelBalance.RemoteBalance.Msat,
		channelBalance.UnsettledLocalBalance.Sat,
		channelBalance.UnsettledLocalBalance.Msat,
		channelBalance.UnsettledRemoteBalance.Sat,
		channelBalance.UnsettledRemoteBalance.Msat,
	)), nil
}

// chainNames extracts chain names from Chain slice.
func chainNames(chains []*lnrpc.Chain) []string {
	names := make([]string, len(chains))
	for i, chain := range chains {
		names[i] = chain.Chain
	}
	return names
}
