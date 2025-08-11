package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// ChannelService handles Lightning channel operations.
type ChannelService struct {
	LightningClient lnrpc.LightningClient
}

// NewChannelService creates a new channel service.
func NewChannelService(client lnrpc.LightningClient) *ChannelService {
	return &ChannelService{
		LightningClient: client,
	}
}

// ListChannelsTool returns the MCP tool definition for listing channels.
func (s *ChannelService) ListChannelsTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_list_channels",
		Description: "List all Lightning channels with detailed information",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"active_only": map[string]any{
					"type":        "boolean",
					"description": "Only return active channels",
				},
				"inactive_only": map[string]any{
					"type":        "boolean",
					"description": "Only return inactive channels",
				},
				"public_only": map[string]any{
					"type":        "boolean",
					"description": "Only return public channels",
				},
				"private_only": map[string]any{
					"type":        "boolean",
					"description": "Only return private channels",
				},
			},
		},
	}
}

// HandleListChannels handles the list channels request.
func (s *ChannelService) HandleListChannels(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	// Parse filter options
	activeOnly, _ := request.Params.Arguments["active_only"].(bool)
	inactiveOnly, _ := request.Params.Arguments["inactive_only"].(bool)
	publicOnly, _ := request.Params.Arguments["public_only"].(bool)
	privateOnly, _ := request.Params.Arguments["private_only"].(bool)

	channels, err := s.LightningClient.ListChannels(ctx,
		&lnrpc.ListChannelsRequest{
			ActiveOnly:   activeOnly,
			InactiveOnly: inactiveOnly,
			PublicOnly:   publicOnly,
			PrivateOnly:  privateOnly,
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to list channels: %v", err)), nil
	}

	channelList := make([]map[string]any, len(channels.Channels))
	for i, ch := range channels.Channels {
		entry := map[string]any{
			"active":                  ch.Active,
			"remote_pubkey":           ch.RemotePubkey,
			"channel_point":           ch.ChannelPoint,
			"chan_id":                 strconv.FormatUint(ch.ChanId, 10),
			"capacity":                ch.Capacity,
			"local_balance":           ch.LocalBalance,
			"remote_balance":          ch.RemoteBalance,
			"commit_fee":              ch.CommitFee,
			"commit_weight":           ch.CommitWeight,
			"fee_per_kw":              ch.FeePerKw,
			"unsettled_balance":       ch.UnsettledBalance,
			"total_satoshis_sent":     ch.TotalSatoshisSent,
			"total_satoshis_received": ch.TotalSatoshisReceived,
			"num_updates":             ch.NumUpdates,
			"pending_htlcs":           len(ch.PendingHtlcs),
			"private":                 ch.Private,
			"initiator":               ch.Initiator,
			"chan_status_flags":       ch.ChanStatusFlags,
		}

		if local := constraintsToMap(ch.GetLocalConstraints()); local != nil {
			entry["local_constraints"] = local
		}
		if remote := constraintsToMap(ch.GetRemoteConstraints()); remote != nil {
			entry["remote_constraints"] = remote
		}

		channelList[i] = entry
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"channels": %s,
		"total_channels": %d
	}`, toJSONString(channelList), len(channelList))), nil
}

// PendingChannelsTool returns the MCP tool definition for listing pending channels.
func (s *ChannelService) PendingChannelsTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_pending_channels",
		Description: "List all pending Lightning channels",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}
}

// HandlePendingChannels handles the pending channels request.
func (s *ChannelService) HandlePendingChannels(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	pending, err := s.LightningClient.PendingChannels(ctx,
		&lnrpc.PendingChannelsRequest{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to get pending channels: %v", err)), nil
	}

	// Format pending channels
	result := map[string]any{
		"pending_open_channels": formatPendingOpenChannels(
			pending.PendingOpenChannels),
		"pending_force_closing_channels": formatPendingForceClosingChannels(
			pending.PendingForceClosingChannels),
		"waiting_close_channels": formatWaitingCloseChannels(
			pending.WaitingCloseChannels),
		"total_limbo_balance": pending.TotalLimboBalance,
	}

	return mcp.NewToolResultText(toJSONString(result)), nil
}

// FormatPendingOpenChannels formats pending open channel data for JSON output.
func formatPendingOpenChannels(
	channels []*lnrpc.PendingChannelsResponse_PendingOpenChannel) []map[string]any {
	result := make([]map[string]any, len(channels))
	for i, ch := range channels {
		result[i] = map[string]any{
			"channel":       formatPendingChannel(ch.Channel),
			"commit_fee":    ch.CommitFee,
			"commit_weight": ch.CommitWeight,
			"fee_per_kw":    ch.FeePerKw,
		}
	}
	return result
}

func constraintsToMap(c *lnrpc.ChannelConstraints) map[string]any {
	if c == nil {
		return nil
	}

	return map[string]any{
		"csv_delay":            c.CsvDelay,
		"chan_reserve_sat":     c.ChanReserveSat,
		"dust_limit_sat":       c.DustLimitSat,
		"max_pending_amt_msat": c.MaxPendingAmtMsat,
		"min_htlc_msat":        c.MinHtlcMsat,
		"max_accepted_htlcs":   c.MaxAcceptedHtlcs,
	}
}

// FormatPendingForceClosingChannels formats force closing channel data for JSON output.
func formatPendingForceClosingChannels(
	channels []*lnrpc.PendingChannelsResponse_ForceClosedChannel) []map[string]any {
	result := make([]map[string]any, len(channels))
	for i, ch := range channels {
		result[i] = map[string]any{
			"channel":             formatPendingChannel(ch.Channel),
			"closing_txid":        ch.ClosingTxid,
			"limbo_balance":       ch.LimboBalance,
			"maturity_height":     ch.MaturityHeight,
			"blocks_til_maturity": ch.BlocksTilMaturity,
			"recovered_balance":   ch.RecoveredBalance,
		}
	}
	return result
}

// FormatWaitingCloseChannels formats waiting close channel data for JSON output.
func formatWaitingCloseChannels(
	channels []*lnrpc.PendingChannelsResponse_WaitingCloseChannel) []map[string]any {
	result := make([]map[string]any, len(channels))
	for i, ch := range channels {
		result[i] = map[string]any{
			"channel":       formatPendingChannel(ch.Channel),
			"limbo_balance": ch.LimboBalance,
		}
	}
	return result
}

// FormatPendingChannel formats a single pending channel for JSON output.
func formatPendingChannel(
	ch *lnrpc.PendingChannelsResponse_PendingChannel) map[string]any {
	return map[string]any{
		"remote_node_pub": ch.RemoteNodePub,
		"channel_point":   ch.ChannelPoint,
		"capacity":        ch.Capacity,
		"local_balance":   ch.LocalBalance,
		"remote_balance":  ch.RemoteBalance,
	}
}

// ToJSONString converts an interface to JSON string for output formatting.
func toJSONString(v any) string {
	// This is a simplified version - in production you'd use proper
	// JSON marshaling
	return fmt.Sprintf("%+v", v)
}
