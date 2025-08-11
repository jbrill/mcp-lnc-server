package tools

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// PaymentService handles read-only Lightning payment operations.
type PaymentService struct {
	LightningClient lnrpc.LightningClient
}

// NewPaymentService creates a new payment service for read-only operations.
func NewPaymentService(lightningClient lnrpc.LightningClient) *PaymentService {
	return &PaymentService{
		LightningClient: lightningClient,
	}
}

// ListPaymentsTool returns the MCP tool definition for listing payments.
func (s *PaymentService) ListPaymentsTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_list_payments",
		Description: "List historical Lightning payments made by this node",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"include_incomplete": map[string]any{
					"type":        "boolean",
					"description": "Include incomplete/failed payments",
				},
				"index_offset": map[string]any{
					"type":        "number",
					"description": "Start index for pagination",
					"minimum":     0,
				},
				"max_payments": map[string]any{
					"type":        "number",
					"description": "Maximum number of payments to return",
					"minimum":     1,
					"maximum":     1000,
				},
				"reversed": map[string]any{
					"type":        "boolean",
					"description": "Return payments in reverse chronological order",
				},
			},
		},
	}
}

// HandleListPayments handles the list payments request.
func (s *PaymentService) HandleListPayments(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	// Parse parameters
	includeIncomplete, _ := request.Params.Arguments["include_incomplete"].(bool)
	indexOffset, _ := request.Params.Arguments["index_offset"].(float64)
	maxPayments, _ := request.Params.Arguments["max_payments"].(float64)
	if maxPayments == 0 {
		maxPayments = 100 // Default
	}
	reversed, _ := request.Params.Arguments["reversed"].(bool)

	// List payments
	resp, err := s.LightningClient.ListPayments(ctx, &lnrpc.ListPaymentsRequest{
		IncludeIncomplete: includeIncomplete,
		IndexOffset:       uint64(indexOffset),
		MaxPayments:       uint64(maxPayments),
		Reversed:          reversed,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to list payments: %v", err)), nil
	}

	// Format payment list
	paymentList := make([]map[string]any, len(resp.Payments))
	for i, payment := range resp.Payments {
		paymentList[i] = map[string]any{
			"payment_hash":     payment.PaymentHash,
			"value_sat":        payment.ValueSat,
			"value_msat":       payment.ValueMsat,
			"payment_preimage": payment.PaymentPreimage,
			"payment_request":  payment.PaymentRequest,
			"status":           payment.Status.String(),
			"fee_sat":          payment.FeeSat,
			"fee_msat":         payment.FeeMsat,
			"creation_time_ns": payment.CreationTimeNs,
			"payment_index":    payment.PaymentIndex,
			"failure_reason":   payment.FailureReason.String(),
			"htlc_count":       len(payment.Htlcs),
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"payments": %s,
		"first_index_offset": %d,
		"last_index_offset": %d,
		"total_payments": %d
	}`, toJSONString(paymentList), resp.FirstIndexOffset,
		resp.LastIndexOffset, len(paymentList))), nil
}

// TrackPaymentTool returns the MCP tool definition for tracking a payment.
func (s *PaymentService) TrackPaymentTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_track_payment",
		Description: "Track the status of a Lightning payment by its hash",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"payment_hash": map[string]any{
					"type":        "string",
					"description": "Payment hash to track (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{64}$",
				},
			},
			Required: []string{"payment_hash"},
		},
	}
}

// HandleTrackPayment handles the track payment request.
func (s *PaymentService) HandleTrackPayment(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	paymentHash, ok := request.Params.Arguments["payment_hash"].(string)
	if !ok {
		return mcp.NewToolResultError("payment_hash is required"), nil
	}

	// Validate payment hash format
	if len(paymentHash) != 64 {
		return mcp.NewToolResultError(
			"payment_hash must be a 64-character hex string"), nil
	}

	// For read-only operation, we'll just look up the payment in history
	resp, err := s.LightningClient.ListPayments(ctx, &lnrpc.ListPaymentsRequest{
		IncludeIncomplete: true,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to fetch payment: %v", err)), nil
	}

	// Find the payment with matching hash
	for _, payment := range resp.Payments {
		if payment.PaymentHash == paymentHash {
			return mcp.NewToolResultText(fmt.Sprintf(`{
				"found": true,
				"payment_hash": "%s",
				"status": "%s",
				"value_sat": %d,
				"fee_sat": %d,
				"creation_time_ns": %d,
				"payment_preimage": "%s",
				"failure_reason": "%s"
			}`, payment.PaymentHash, payment.Status.String(),
				payment.ValueSat, payment.FeeSat,
				payment.CreationTimeNs, payment.PaymentPreimage,
				payment.FailureReason.String())), nil
		}
	}

	return mcp.NewToolResultText(`{"found": false, "message": "Payment not found"}`), nil
}

// Helper function to check BOLT11 format
//
//nolint:unused // Used in tests
func isValidBolt11(invoice string) bool {
	// Basic check: BOLT11 invoices start with "ln"
	return len(invoice) > 2 && invoice[:2] == "ln"
}
