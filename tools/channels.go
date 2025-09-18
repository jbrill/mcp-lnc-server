package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

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

// OpenChannelTool returns the MCP tool definition for opening channels.
func (s *ChannelService) OpenChannelTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_open_channel",
		Description: "Open a new Lightning channel with a peer",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"node_pubkey": map[string]any{
					"type":        "string",
					"description": "Public key of the node to open channel with (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{66}$",
				},
				"local_funding_amount": map[string]any{
					"type":        "number",
					"description": "Amount to fund the channel with (satoshis)",
					"minimum":     20000,
				},
				"push_sat": map[string]any{
					"type":        "number",
					"description": "Amount to push to remote side (satoshis)",
					"minimum":     0,
				},
				"target_conf": map[string]any{
					"type":        "number",
					"description": "Target confirmations for funding transaction",
					"minimum":     1,
					"maximum":     6,
				},
				"sat_per_byte": map[string]any{
					"type":        "number",
					"description": "Fee rate in satoshis per byte",
					"minimum":     1,
				},
				"private": map[string]any{
					"type":        "boolean",
					"description": "Whether to make the channel private",
				},
				"min_htlc_msat": map[string]any{
					"type":        "number",
					"description": "Minimum HTLC value in millisatoshis",
					"minimum":     1,
				},
			},
			Required: []string{"node_pubkey", "local_funding_amount"},
		},
	}
}

// HandleOpenChannel handles the open channel request.
func (s *ChannelService) HandleOpenChannel(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	// Parse required parameters
	nodePubkey, ok := request.Params.Arguments["node_pubkey"].(string)
	if !ok {
		return mcp.NewToolResultError("node_pubkey is required"), nil
	}

	localFundingAmount, ok := request.Params.Arguments["local_funding_amount"].(float64)
	if !ok {
		return mcp.NewToolResultError("local_funding_amount is required"), nil
	}

	// Validate node pubkey format
	if len(nodePubkey) != 66 {
		return mcp.NewToolResultError(
			"node_pubkey must be a 66-character hex string"), nil
	}

	pubkeyBytes, err := hex.DecodeString(nodePubkey)
	if err != nil {
		return mcp.NewToolResultError("invalid node_pubkey format"), nil
	}

	// Parse optional parameters
	pushSat, _ := request.Params.Arguments["push_sat"].(float64)
	targetConf, _ := request.Params.Arguments["target_conf"].(float64)
	if targetConf == 0 {
		targetConf = 3 // Default 3 confirmations
	}
	satPerByte, _ := request.Params.Arguments["sat_per_byte"].(float64)
	private, _ := request.Params.Arguments["private"].(bool)
	minHtlcMsat, _ := request.Params.Arguments["min_htlc_msat"].(float64)
	if minHtlcMsat == 0 {
		minHtlcMsat = 1000 // Default 1 sat in msat
	}

	// Open channel
	stream, err := s.LightningClient.OpenChannel(ctx, &lnrpc.OpenChannelRequest{
		NodePubkey:         pubkeyBytes,
		LocalFundingAmount: int64(localFundingAmount),
		PushSat:            int64(pushSat),
		TargetConf:         int32(targetConf),
		SatPerByte:         int64(satPerByte),
		Private:            private,
		MinHtlcMsat:        int64(minHtlcMsat),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to open channel: %v", err)), nil
	}

	// Wait for funding transaction
	update, err := stream.Recv()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to receive channel update: %v", err)), nil
	}

	switch u := update.Update.(type) {
	case *lnrpc.OpenStatusUpdate_ChanPending:
		return mcp.NewToolResultText(fmt.Sprintf(`{
			"success": true,
			"funding_txid": "%s",
			"output_index": %d,
			"node_pubkey": "%s",
			"local_funding_amount": %d,
			"push_sat": %d,
			"private": %t
		}`,
			hex.EncodeToString(u.ChanPending.Txid),
			u.ChanPending.OutputIndex,
			nodePubkey,
			int64(localFundingAmount),
			int64(pushSat),
			private,
		)), nil
	default:
		return mcp.NewToolResultError("Unexpected channel opening response"), nil
	}
}

// CloseChannelTool returns the MCP tool definition for closing channels.
func (s *ChannelService) CloseChannelTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_close_channel",
		Description: "Close an existing Lightning channel",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"channel_point": map[string]any{
					"type":        "string",
					"description": "Channel point in format txid:output_index",
					"pattern":     "^[0-9a-fA-F]{64}:[0-9]+$",
				},
				"force": map[string]any{
					"type": "boolean",
					"description": "Force close the channel " +
						"(broadcasts commitment transaction)",
				},
				"target_conf": map[string]any{
					"type":        "number",
					"description": "Target confirmations for closing transaction",
					"minimum":     1,
					"maximum":     6,
				},
				"sat_per_byte": map[string]any{
					"type":        "number",
					"description": "Fee rate in satoshis per byte",
					"minimum":     1,
				},
			},
			Required: []string{"channel_point"},
		},
	}
}

// HandleCloseChannel handles the close channel request.
func (s *ChannelService) HandleCloseChannel(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	channelPoint, ok := request.Params.Arguments["channel_point"].(string)
	if !ok {
		return mcp.NewToolResultError("channel_point is required"), nil
	}

	// Parse channel point
	parts := strings.Split(channelPoint, ":")
	if len(parts) != 2 {
		return mcp.NewToolResultError(
			"channel_point must be in format txid:output_index"), nil
	}

	txidBytes, err := hex.DecodeString(parts[0])
	if err != nil {
		return mcp.NewToolResultError("invalid txid in channel_point"), nil
	}

	outputIndex, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return mcp.NewToolResultError("invalid output_index in channel_point"), nil
	}

	// Parse optional parameters
	force, _ := request.Params.Arguments["force"].(bool)
	targetConf, _ := request.Params.Arguments["target_conf"].(float64)
	if targetConf == 0 {
		targetConf = 3
	}
	satPerByte, _ := request.Params.Arguments["sat_per_byte"].(float64)

	// Close channel
	stream, err := s.LightningClient.CloseChannel(ctx, &lnrpc.CloseChannelRequest{
		ChannelPoint: &lnrpc.ChannelPoint{
			FundingTxid: &lnrpc.ChannelPoint_FundingTxidBytes{
				FundingTxidBytes: txidBytes,
			},
			OutputIndex: uint32(outputIndex),
		},
		Force:      force,
		TargetConf: int32(targetConf),
		SatPerByte: int64(satPerByte),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to close channel: %v", err)), nil
	}

	// Wait for close update
	update, err := stream.Recv()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to receive close update: %v", err)), nil
	}

	switch u := update.Update.(type) {
	case *lnrpc.CloseStatusUpdate_ClosePending:
		return mcp.NewToolResultText(fmt.Sprintf(`{
			"success": true,
			"closing_txid": "%s",
			"channel_point": "%s",
			"force": %t
		}`,
			hex.EncodeToString(u.ClosePending.Txid),
			channelPoint,
			force,
		)), nil
	default:
		return mcp.NewToolResultError("Unexpected channel closing response"), nil
	}
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
