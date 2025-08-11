package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	lnccontext "github.com/jbrill/mcp-lnc-server/internal/context"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// InvoiceService handles Lightning invoice operations.
type InvoiceService struct {
	LightningClient lnrpc.LightningClient
}

// NewInvoiceService creates a new invoice service.
func NewInvoiceService(client lnrpc.LightningClient) *InvoiceService {
	return &InvoiceService{
		LightningClient: client,
	}
}

// CreateInvoiceTool returns the MCP tool definition for creating invoices.
func (s *InvoiceService) CreateInvoiceTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_create_invoice",
		Description: "Create a Lightning Network invoice for receiving payments",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"amount": map[string]any{
					"type":        "number",
					"description": "Amount in satoshis",
					"minimum":     1,
				},
				"memo": map[string]any{
					"type":        "string",
					"description": "Optional memo/description for the invoice",
					"maxLength":   1024,
				},
				"expiry": map[string]any{
					"type":        "number",
					"description": "Expiry time in seconds (default: 3600 = 1 hour)",
					"minimum":     60,
					"maximum":     31536000, // 1 year
				},
				"private": map[string]any{
					"type":        "boolean",
					"description": "Whether to include routing hints for private channels",
				},
			},
			Required: []string{"amount"},
		},
	}
}

// HandleCreateInvoice handles the create invoice request.
func (s *InvoiceService) HandleCreateInvoice(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create request context with tracing
	reqCtx := lnccontext.New(ctx, "create_invoice", 30*time.Second)
	logger := logging.LogWithContext(reqCtx)
	
	logger.Info("Creating Lightning invoice",
		zap.Any("params", request.Params.Arguments))
	
	if s.LightningClient == nil {
		logger.Error("Lightning client not available")
		return mcp.NewToolResultError(
			"Not connected to Lightning node. " +
				"Use lnc_connect first."), nil
	}

	// Parse and validate amount
	amount, ok := request.Params.Arguments["amount"].(float64)
	if !ok {
		return mcp.NewToolResultError("amount is required and must be a number"), nil
	}
	if amount < 1 {
		return mcp.NewToolResultError("amount must be at least 1 satoshi"), nil
	}

	// Parse optional parameters
	memo, _ := request.Params.Arguments["memo"].(string)
	if len(memo) > 1024 {
		return mcp.NewToolResultError("memo cannot exceed 1024 characters"), nil
	}

	expiry, _ := request.Params.Arguments["expiry"].(float64)
	if expiry == 0 {
		expiry = 3600 // Default 1 hour
	}
	if expiry < 60 || expiry > 31536000 {
		return mcp.NewToolResultError(
			"expiry must be between 60 seconds and 1 year"), nil
	}

	private, _ := request.Params.Arguments["private"].(bool)

	// Create invoice
	invoice, err := s.LightningClient.AddInvoice(reqCtx, &lnrpc.Invoice{
		Value:   int64(amount),
		Memo:    memo,
		Expiry:  int64(expiry),
		Private: private,
	})
	if err != nil {
		logger.Error("Failed to create invoice", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to create invoice: %v", err)), nil
	}
	
	logger.Info("Invoice created successfully",
		zap.String("payment_hash", hex.EncodeToString(invoice.RHash)))

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"payment_request": "%s",
		"payment_hash": "%s",
		"amount_sats": %d,
		"memo": "%s",
		"expiry_seconds": %d,
		"private": %t,
		"r_hash": "%s",
		"add_index": %d
	}`,
		invoice.PaymentRequest,
		hex.EncodeToString(invoice.RHash),
		int64(amount),
		memo,
		int64(expiry),
		private,
		hex.EncodeToString(invoice.RHash),
		invoice.AddIndex,
	)), nil
}

// DecodeInvoiceTool returns the MCP tool definition for decoding invoices.
func (s *InvoiceService) DecodeInvoiceTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_decode_invoice",
		Description: "Decode a BOLT11 Lightning invoice to inspect its contents",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"invoice": map[string]any{
					"type":        "string",
					"description": "BOLT11 invoice string to decode",
					"pattern":     "^ln[a-z0-9]+$",
				},
			},
			Required: []string{"invoice"},
		},
	}
}

// HandleDecodeInvoice handles the decode invoice request.
func (s *InvoiceService) HandleDecodeInvoice(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create request context with tracing
	reqCtx := lnccontext.New(ctx, "decode_invoice", 15*time.Second)
	logger := logging.LogWithContext(reqCtx)
	
	logger.Info("Decoding Lightning invoice")
	
	if s.LightningClient == nil {
		logger.Error("Lightning client not available")
		return mcp.NewToolResultError(
			"Not connected to Lightning node. " +
				"Use lnc_connect first."), nil
	}

	invoice, ok := request.Params.Arguments["invoice"].(string)
	if !ok {
		return mcp.NewToolResultError("invoice is required"), nil
	}

	// Basic validation
	if !isValidBolt11(invoice) {
		return mcp.NewToolResultError("invalid BOLT11 invoice format"), nil
	}

	// Decode invoice
	payReq, err := s.LightningClient.DecodePayReq(reqCtx, &lnrpc.PayReqString{
		PayReq: invoice,
	})
	if err != nil {
		logger.Error("Failed to decode invoice", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to decode invoice: %v", err)), nil
	}
	
	logger.Info("Invoice decoded successfully",
		zap.String("destination", payReq.Destination))

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"destination": "%s",
		"payment_hash": "%s",
		"num_satoshis": %d,
		"num_msat": %d,
		"timestamp": %d,
		"expiry": %d,
		"description": "%s",
		"description_hash": "%s",
		"fallback_addr": "%s",
		"cltv_expiry": %d,
		"route_hints": %d
	}`,
		payReq.Destination,
		payReq.PaymentHash,
		payReq.NumSatoshis,
		payReq.NumMsat,
		payReq.Timestamp,
		payReq.Expiry,
		payReq.Description,
		payReq.DescriptionHash,
		payReq.FallbackAddr,
		payReq.CltvExpiry,
		len(payReq.RouteHints),
	)), nil
}

// IsValidBolt11 performs basic BOLT11 format validation.
func isValidBolt11(invoice string) bool {
	if len(invoice) < 10 {
		return false
	}
	// BOLT11 invoices start with "ln" followed by network prefix
	return invoice[:2] == "ln"
}
