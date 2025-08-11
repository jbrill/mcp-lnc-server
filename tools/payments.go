package tools

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// PaymentService handles Lightning payment operations.
type PaymentService struct {
	LightningClient lnrpc.LightningClient
	RouterClient    routerrpc.RouterClient
}

// NewPaymentService creates a new payment service.
func NewPaymentService(lightningClient lnrpc.LightningClient,
	routerClient routerrpc.RouterClient) *PaymentService {
	return &PaymentService{
		LightningClient: lightningClient,
		RouterClient:    routerClient,
	}
}

// PayInvoiceTool returns the MCP tool definition for paying invoices.
func (s *PaymentService) PayInvoiceTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_pay_invoice",
		Description: "Pay a BOLT11 Lightning invoice",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"invoice": map[string]any{
					"type":        "string",
					"description": "BOLT11 invoice string to pay",
					"pattern":     "^ln[a-z0-9]+$",
				},
				"max_fee_sats": map[string]any{
					"type":        "number",
					"description": "Maximum fee willing to pay in satoshis (default: 1000)",
					"minimum":     0,
					"maximum":     1000000,
				},
				"timeout_seconds": map[string]any{
					"type":        "number",
					"description": "Payment timeout in seconds (default: 60)",
					"minimum":     10,
					"maximum":     3600,
				},
				"allow_self_payment": map[string]any{
					"type":        "boolean",
					"description": "Allow paying invoices created by this node",
				},
			},
			Required: []string{"invoice"},
		},
	}
}

// HandlePayInvoice handles the pay invoice request.
func (s *PaymentService) HandlePayInvoice(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.RouterClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	invoice, ok := request.Params.Arguments["invoice"].(string)
	if !ok {
		return mcp.NewToolResultError("invoice is required"), nil
	}

	// Basic validation
	if !isValidBolt11(invoice) {
		return mcp.NewToolResultError("invalid BOLT11 invoice format"), nil
	}

	// Parse optional parameters
	maxFeeSats, _ := request.Params.Arguments["max_fee_sats"].(float64)
	if maxFeeSats == 0 {
		maxFeeSats = 1000 // Default max fee
	}

	timeoutSeconds, _ := request.Params.Arguments["timeout_seconds"].(float64)
	if timeoutSeconds == 0 {
		timeoutSeconds = 60 // Default timeout
	}

	allowSelfPayment, _ := request.Params.Arguments["allow_self_payment"].(bool)

	// Send payment using router
	stream, err := s.RouterClient.SendPaymentV2(ctx, &routerrpc.SendPaymentRequest{
		PaymentRequest:   invoice,
		FeeLimitSat:      int64(maxFeeSats),
		TimeoutSeconds:   int32(timeoutSeconds),
		AllowSelfPayment: allowSelfPayment,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to initiate payment: %v", err)), nil
	}

	// Wait for payment result
	for {
		update, err := stream.Recv()
		if err != nil {
			return mcp.NewToolResultError(
				fmt.Sprintf("Error receiving payment update: %v", err)), nil
		}

		switch update.Status {
		case lnrpc.Payment_SUCCEEDED:
			return mcp.NewToolResultText(fmt.Sprintf(`{
				"success": true,
				"payment_preimage": "%s",
				"payment_hash": "%s",
				"fee_sat": %d,
				"fee_msat": %d,
				"value_sat": %d,
				"value_msat": %d,
				"payment_index": %d,
				"creation_time_ns": %d,
				"htlcs": %d
			}`,
				update.PaymentPreimage,
				update.PaymentHash,
				update.FeeSat,
				update.FeeMsat,
				update.ValueSat,
				update.ValueMsat,
				update.PaymentIndex,
				update.CreationTimeNs,
				len(update.Htlcs),
			)), nil

		case lnrpc.Payment_FAILED:
			return mcp.NewToolResultError(fmt.Sprintf(`Payment failed: %s. Details: %s`,
				update.FailureReason.String(),
				getFailureDetails(update),
			)), nil

		case lnrpc.Payment_IN_FLIGHT:
			// Continue waiting
			continue

		default:
			return mcp.NewToolResultError(
				fmt.Sprintf("Unknown payment status: %v", update.Status)), nil
		}
	}
}

// SendPaymentTool returns the MCP tool definition for sending keysend payments.
func (s *PaymentService) SendPaymentTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_send_payment",
		Description: "Send a spontaneous payment (keysend) to a Lightning node",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"destination": map[string]any{
					"type":        "string",
					"description": "Public key of the destination node (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{66}$",
				},
				"amount_sats": map[string]any{
					"type":        "number",
					"description": "Amount to send in satoshis",
					"minimum":     1,
				},
				"max_fee_sats": map[string]any{
					"type":        "number",
					"description": "Maximum fee willing to pay in satoshis (default: 1000)",
					"minimum":     0,
				},
				"timeout_seconds": map[string]any{
					"type":        "number",
					"description": "Payment timeout in seconds (default: 60)",
					"minimum":     10,
					"maximum":     3600,
				},
			},
			Required: []string{"destination", "amount_sats"},
		},
	}
}

// HandleSendPayment handles the send payment request.
func (s *PaymentService) HandleSendPayment(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.RouterClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	destination, ok := request.Params.Arguments["destination"].(string)
	if !ok {
		return mcp.NewToolResultError("destination is required"), nil
	}

	amountSats, ok := request.Params.Arguments["amount_sats"].(float64)
	if !ok {
		return mcp.NewToolResultError("amount_sats is required"), nil
	}

	// Validate destination format (66 char hex string)
	if len(destination) != 66 {
		return mcp.NewToolResultError(
			"destination must be a 66-character hex-encoded public key"), nil
	}

	// Parse optional parameters
	maxFeeSats, _ := request.Params.Arguments["max_fee_sats"].(float64)
	if maxFeeSats == 0 {
		maxFeeSats = 1000
	}

	timeoutSeconds, _ := request.Params.Arguments["timeout_seconds"].(float64)
	if timeoutSeconds == 0 {
		timeoutSeconds = 60
	}

	// Send keysend payment
	stream, err := s.RouterClient.SendPaymentV2(ctx, &routerrpc.SendPaymentRequest{
		Dest:           []byte(destination),
		Amt:            int64(amountSats),
		FeeLimitSat:    int64(maxFeeSats),
		TimeoutSeconds: int32(timeoutSeconds),
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to initiate keysend payment: %v", err)), nil
	}

	// Wait for payment result (same logic as PayInvoice)
	for {
		update, err := stream.Recv()
		if err != nil {
			return mcp.NewToolResultError(
				fmt.Sprintf("Error receiving payment update: %v", err)), nil
		}

		switch update.Status {
		case lnrpc.Payment_SUCCEEDED:
			return mcp.NewToolResultText(fmt.Sprintf(`{
				"success": true,
				"payment_preimage": "%s",
				"payment_hash": "%s",
				"destination": "%s",
				"fee_sat": %d,
				"value_sat": %d,
				"payment_index": %d
			}`,
				update.PaymentPreimage,
				update.PaymentHash,
				destination,
				update.FeeSat,
				update.ValueSat,
				update.PaymentIndex,
			)), nil

		case lnrpc.Payment_FAILED:
			return mcp.NewToolResultError(
				fmt.Sprintf("Keysend payment failed: %s",
					update.FailureReason.String())), nil

		case lnrpc.Payment_IN_FLIGHT:
			continue

		default:
			return mcp.NewToolResultError(
				fmt.Sprintf("Unknown payment status: %v", update.Status)), nil
		}
	}
}

// getFailureDetails extracts useful failure information from a payment update.
func getFailureDetails(update *lnrpc.Payment) string {
	if len(update.Htlcs) == 0 {
		return "No HTLC details available"
	}

	// Get the last HTLC attempt for failure details
	lastHtlc := update.Htlcs[len(update.Htlcs)-1]
	if lastHtlc.Failure != nil {
		return fmt.Sprintf("Code: %v, Source: %d", lastHtlc.Failure.Code,
			lastHtlc.Failure.FailureSourceIndex)
	}

	return "No specific failure details available"
}
