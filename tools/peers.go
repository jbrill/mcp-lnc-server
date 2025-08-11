package tools

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// PeerService handles Lightning peer operations.
type PeerService struct {
	LightningClient lnrpc.LightningClient
}

// NewPeerService creates a new peer service.
func NewPeerService(client lnrpc.LightningClient) *PeerService {
	return &PeerService{
		LightningClient: client,
	}
}

// ListPeersTool returns the MCP tool definition for listing peers.
func (s *PeerService) ListPeersTool() mcp.Tool {
	return mcp.Tool{
		Name: "lnc_list_peers",
		Description: "List all connected Lightning Network peers with " +
			"detailed connection information",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}
}

// HandleListPeers handles the list peers request.
func (s *PeerService) HandleListPeers(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	peers, err := s.LightningClient.ListPeers(ctx, &lnrpc.ListPeersRequest{})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to list peers: %v", err)), nil
	}

	peerList := make([]map[string]any, len(peers.Peers))
	for i, peer := range peers.Peers {
		// Format peer features
		features := make([]map[string]any, 0)
		for featureKey, feature := range peer.Features {
			features = append(features, map[string]any{
				"feature":     featureKey,
				"name":        feature.Name,
				"is_required": feature.IsRequired,
				"is_known":    feature.IsKnown,
			})
		}

		// Format error information (simplified)
		var lastError map[string]any
		if len(peer.Errors) > 0 {
			lastError = map[string]any{
				"last_error": peer.Errors[len(peer.Errors)-1].Error,
			}
		}

		peerList[i] = map[string]any{
			"pub_key":    peer.PubKey,
			"address":    peer.Address,
			"bytes_sent": peer.BytesSent,
			"bytes_recv": peer.BytesRecv,
			"sat_sent":   peer.SatSent,
			"sat_recv":   peer.SatRecv,
			"inbound":    peer.Inbound,
			"ping_time":  peer.PingTime,
			"sync_type":  peer.SyncType.String(),
			"features":   features,
			"errors":     formatPeerErrors(peer.Errors),
			"flap_count": peer.FlapCount,
			"last_flap":  lastError,
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"peers": %s,
		"total_peers": %d
	}`, toJSONStringPeers(peerList), len(peerList))), nil
}

// ConnectPeerTool returns the MCP tool definition for connecting to peers.
func (s *PeerService) ConnectPeerTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_connect_peer",
		Description: "Connect to a Lightning Network peer",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"node_pubkey": map[string]any{
					"type":        "string",
					"description": "Public key of the peer to connect to (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{66}$",
				},
				"address": map[string]any{
					"type":        "string",
					"description": "Network address of the peer (host:port)",
					"pattern":     "^.+:[0-9]+$",
				},
				"permanent": map[string]any{
					"type":        "boolean",
					"description": "Whether to make this a permanent connection",
				},
				"timeout": map[string]any{
					"type":        "number",
					"description": "Connection timeout in seconds (default: 10)",
					"minimum":     1,
					"maximum":     300,
				},
			},
			Required: []string{"node_pubkey", "address"},
		},
	}
}

// HandleConnectPeer handles the connect peer request.
func (s *PeerService) HandleConnectPeer(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	nodePubkey, ok := request.Params.Arguments["node_pubkey"].(string)
	if !ok {
		return mcp.NewToolResultError("node_pubkey is required"), nil
	}

	address, ok := request.Params.Arguments["address"].(string)
	if !ok {
		return mcp.NewToolResultError("address is required"), nil
	}

	// Validate node pubkey format
	if len(nodePubkey) != 66 {
		return mcp.NewToolResultError(
			"node_pubkey must be a 66-character hex string"), nil
	}

	// Parse optional parameters
	permanent, _ := request.Params.Arguments["permanent"].(bool)
	timeout, _ := request.Params.Arguments["timeout"].(float64)
	if timeout == 0 {
		timeout = 10 // Default 10 seconds
	}

	// Connect to peer
	_, err := s.LightningClient.ConnectPeer(ctx, &lnrpc.ConnectPeerRequest{
		Addr: &lnrpc.LightningAddress{
			Pubkey: nodePubkey,
			Host:   address,
		},
		Perm:    permanent,
		Timeout: uint64(timeout),
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to connect to peer: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"success": true,
		"connected_to": "%s",
		"address": "%s",
		"permanent": %t
	}`, nodePubkey, address, permanent)), nil
}

// DisconnectPeerTool returns the MCP tool definition for disconnecting peers.
func (s *PeerService) DisconnectPeerTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_disconnect_peer",
		Description: "Disconnect from a Lightning Network peer",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"node_pubkey": map[string]any{
					"type":        "string",
					"description": "Public key of the peer to disconnect from (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{66}$",
				},
			},
			Required: []string{"node_pubkey"},
		},
	}
}

// HandleDisconnectPeer handles the disconnect peer request.
func (s *PeerService) HandleDisconnectPeer(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	nodePubkey, ok := request.Params.Arguments["node_pubkey"].(string)
	if !ok {
		return mcp.NewToolResultError("node_pubkey is required"), nil
	}

	// Validate node pubkey format
	if len(nodePubkey) != 66 {
		return mcp.NewToolResultError(
			"node_pubkey must be a 66-character hex string"), nil
	}

	// Disconnect from peer
	_, err := s.LightningClient.DisconnectPeer(ctx, &lnrpc.DisconnectPeerRequest{
		PubKey: nodePubkey,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to disconnect from peer: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"success": true,
		"disconnected_from": "%s"
	}`, nodePubkey)), nil
}

// DescribeGraphTool returns the MCP tool definition for getting network graph.
func (s *PeerService) DescribeGraphTool() mcp.Tool {
	return mcp.Tool{
		Name: "lnc_describe_graph",
		Description: "Get Lightning Network graph information including " +
			"nodes and channels",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"include_unannounced": map[string]any{
					"type":        "boolean",
					"description": "Include unannounced channels in the graph",
				},
			},
		},
	}
}

// HandleDescribeGraph handles the describe graph request.
func (s *PeerService) HandleDescribeGraph(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	includeUnannounced, _ := request.Params.Arguments["include_unannounced"].(bool)

	graph, err := s.LightningClient.DescribeGraph(ctx, &lnrpc.ChannelGraphRequest{
		IncludeUnannounced: includeUnannounced,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to describe graph: %v", err)), nil
	}

	// Format the graph data (simplified for readability)
	nodeCount := len(graph.Nodes)
	edgeCount := len(graph.Edges)

	// Sample of first few nodes and edges to avoid overwhelming output
	maxSamples := 5
	sampleNodes := make([]map[string]any, 0)
	for i, node := range graph.Nodes {
		if i >= maxSamples {
			break
		}

		addresses := make([]string, len(node.Addresses))
		for j, addr := range node.Addresses {
			addresses[j] = addr.Addr // Just the address without port for now
		}

		sampleNodes = append(sampleNodes, map[string]any{
			"pub_key":     node.PubKey,
			"alias":       node.Alias,
			"addresses":   addresses,
			"color":       node.Color,
			"last_update": node.LastUpdate,
		})
	}

	sampleEdges := make([]map[string]any, 0)
	for i, edge := range graph.Edges {
		if i >= maxSamples {
			break
		}

		sampleEdges = append(sampleEdges, map[string]any{
			"channel_id":  edge.ChannelId,
			"chan_point":  edge.ChanPoint,
			"node1_pub":   edge.Node1Pub,
			"node2_pub":   edge.Node2Pub,
			"capacity":    edge.Capacity,
			"last_update": edge.LastUpdate,
		})
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"total_nodes": %d,
		"total_edges": %d,
		"include_unannounced": %t,
		"sample_nodes": %s,
		"sample_edges": %s
	}`,
		nodeCount,
		edgeCount,
		includeUnannounced,
		toJSONStringPeers(sampleNodes),
		toJSONStringPeers(sampleEdges),
	)), nil
}

// GetNodeInfoTool returns the MCP tool definition for getting specific node information.
func (s *PeerService) GetNodeInfoTool() mcp.Tool {
	return mcp.Tool{
		Name: "lnc_get_node_info",
		Description: "Get detailed information about a specific " +
			"Lightning Network node",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"pub_key": map[string]any{
					"type":        "string",
					"description": "Public key of the node to get info for (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{66}$",
				},
				"include_channels": map[string]any{
					"type":        "boolean",
					"description": "Include the node's channels in the response",
				},
			},
			Required: []string{"pub_key"},
		},
	}
}

// HandleGetNodeInfo handles the get node info request.
func (s *PeerService) HandleGetNodeInfo(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	pubKey, ok := request.Params.Arguments["pub_key"].(string)
	if !ok {
		return mcp.NewToolResultError("pub_key is required"), nil
	}

	includeChannels, _ := request.Params.Arguments["include_channels"].(bool)

	nodeInfo, err := s.LightningClient.GetNodeInfo(ctx, &lnrpc.NodeInfoRequest{
		PubKey:          pubKey,
		IncludeChannels: includeChannels,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to get node info: %v", err)), nil
	}

	// Format node information
	addresses := make([]string, len(nodeInfo.Node.Addresses))
	for i, addr := range nodeInfo.Node.Addresses {
		addresses[i] = addr.Addr // Just the address without port for now
	}

	nodeData := map[string]any{
		"pub_key":        nodeInfo.Node.PubKey,
		"alias":          nodeInfo.Node.Alias,
		"addresses":      addresses,
		"color":          nodeInfo.Node.Color,
		"last_update":    nodeInfo.Node.LastUpdate,
		"num_channels":   nodeInfo.NumChannels,
		"total_capacity": nodeInfo.TotalCapacity,
	}

	if includeChannels && len(nodeInfo.Channels) > 0 {
		channels := make([]map[string]any, len(nodeInfo.Channels))
		for i, channel := range nodeInfo.Channels {
			channels[i] = map[string]any{
				"channel_id":  channel.ChannelId,
				"chan_point":  channel.ChanPoint,
				"node1_pub":   channel.Node1Pub,
				"node2_pub":   channel.Node2Pub,
				"capacity":    channel.Capacity,
				"last_update": channel.LastUpdate,
			}
		}
		nodeData["channels"] = channels
	}

	return mcp.NewToolResultText(toJSONStringPeers(nodeData)), nil
}

// formatPeerErrors formats peer error information for JSON output.
func formatPeerErrors(errors []*lnrpc.TimestampedError,
) []map[string]any {
	result := make([]map[string]any, len(errors))
	for i, err := range errors {
		result[i] = map[string]any{
			"error":     err.Error,
			"timestamp": err.Timestamp,
		}
	}
	return result
}

// toJSONStringPeers converts an interface to JSON string for peer data output.
func toJSONStringPeers(v any) string {
	// Simplified JSON conversion - in production use proper JSON marshaling
	return fmt.Sprintf("%+v", v)
}
