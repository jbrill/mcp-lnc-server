package tools

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/mark3labs/mcp-go/mcp"
)

// InvoiceService handles read-only Lightning invoice operations.
type InvoiceService struct {
	LightningClient lnrpc.LightningClient
}

// NewInvoiceService creates a new invoice service for read-only operations.
func NewInvoiceService(client lnrpc.LightningClient) *InvoiceService {
	return &InvoiceService{
		LightningClient: client,
	}
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
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	invoice, ok := request.Params.Arguments["invoice"].(string)
	if !ok {
		return mcp.NewToolResultError("invoice is required"), nil
	}

	// Basic validation
	if len(invoice) < 3 || invoice[:2] != "ln" {
		return mcp.NewToolResultError("invalid BOLT11 invoice format"), nil
	}

	// Decode the invoice
	decoded, err := s.LightningClient.DecodePayReq(ctx, &lnrpc.PayReqString{
		PayReq: invoice,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to decode invoice: %v", err)), nil
	}

	// Format route hints if present
	routeHints := make([]map[string]any, len(decoded.RouteHints))
	for i, hint := range decoded.RouteHints {
		hops := make([]map[string]any, len(hint.HopHints))
		for j, hop := range hint.HopHints {
			hops[j] = map[string]any{
				"node_id":    hop.NodeId,
				"chan_id":    hop.ChanId,
				"fee_base":   hop.FeeBaseMsat,
				"fee_prop":   hop.FeeProportionalMillionths,
				"cltv_delta": hop.CltvExpiryDelta,
			}
		}
		routeHints[i] = map[string]any{
			"hop_hints": hops,
		}
	}

	// Format features if present
	features := make(map[string]bool)
	for k, v := range decoded.Features {
		features[fmt.Sprintf("%d", k)] = v.IsKnown
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"destination": "%s",
		"payment_hash": "%s",
		"amount_sats": %d,
		"amount_msat": %d,
		"timestamp": %d,
		"expiry": %d,
		"description": "%s",
		"description_hash": "%s",
		"fallback_address": "%s",
		"cltv_expiry": %d,
		"route_hints": %s,
		"payment_addr": "%s",
		"features": %s
	}`,
		decoded.Destination,
		decoded.PaymentHash,
		decoded.NumSatoshis,
		decoded.NumMsat,
		decoded.Timestamp,
		decoded.Expiry,
		decoded.Description,
		decoded.DescriptionHash,
		decoded.FallbackAddr,
		decoded.CltvExpiry,
		toJSONString(routeHints),
		hex.EncodeToString(decoded.PaymentAddr),
		toJSONString(features),
	)), nil
}

// ListInvoicesTool returns the MCP tool definition for listing invoices.
func (s *InvoiceService) ListInvoicesTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_list_invoices",
		Description: "List invoices created by this Lightning node",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"pending_only": map[string]any{
					"type":        "boolean",
					"description": "Only return pending/unpaid invoices",
				},
				"index_offset": map[string]any{
					"type":        "number",
					"description": "Start index for pagination",
					"minimum":     0,
				},
				"num_max_invoices": map[string]any{
					"type":        "number",
					"description": "Maximum number of invoices to return",
					"minimum":     1,
					"maximum":     1000,
				},
				"reversed": map[string]any{
					"type":        "boolean",
					"description": "Return invoices in reverse chronological order",
				},
			},
		},
	}
}

// HandleListInvoices handles the list invoices request.
func (s *InvoiceService) HandleListInvoices(ctx context.Context,
	request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.LightningClient == nil {
		return mcp.NewToolResultError(
			"Not connected to Lightning node. Use lnc_connect first."), nil
	}

	// Parse parameters
	pendingOnly, _ := request.Params.Arguments["pending_only"].(bool)
	indexOffset, _ := request.Params.Arguments["index_offset"].(float64)
	numMaxInvoices, _ := request.Params.Arguments["num_max_invoices"].(float64)
	if numMaxInvoices == 0 {
		numMaxInvoices = 100 // Default
	}
	reversed, _ := request.Params.Arguments["reversed"].(bool)

	// List invoices
	resp, err := s.LightningClient.ListInvoices(ctx, &lnrpc.ListInvoiceRequest{
		PendingOnly:    pendingOnly,
		IndexOffset:    uint64(indexOffset),
		NumMaxInvoices: uint64(numMaxInvoices),
		Reversed:       reversed,
	})
	if err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to list invoices: %v", err)), nil
	}

	// Format invoice list
	invoiceList := make([]map[string]any, len(resp.Invoices))
	for i, invoice := range resp.Invoices {
		invoiceList[i] = map[string]any{
			"memo":            invoice.Memo,
			"payment_request": invoice.PaymentRequest,
			"r_hash":          hex.EncodeToString(invoice.RHash),
			"value":           invoice.Value,
			"value_msat":      invoice.ValueMsat,
			"settled":         invoice.State == lnrpc.Invoice_SETTLED,
			"creation_date":   invoice.CreationDate,
			"settle_date":     invoice.SettleDate,
			"expiry":          invoice.Expiry,
			"cltv_expiry":     invoice.CltvExpiry,
			"private":         invoice.Private,
			"add_index":       invoice.AddIndex,
			"settle_index":    invoice.SettleIndex,
			"amt_paid_sat":    invoice.AmtPaidSat,
			"amt_paid_msat":   invoice.AmtPaidMsat,
			"state":           invoice.State.String(),
			"is_keysend":      invoice.IsKeysend,
			"payment_addr":    hex.EncodeToString(invoice.PaymentAddr),
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"invoices": %s,
		"first_index_offset": %d,
		"last_index_offset": %d,
		"total_invoices": %d
	}`, toJSONString(invoiceList), resp.FirstIndexOffset,
		resp.LastIndexOffset, len(invoiceList))), nil
}

// LookupInvoiceTool returns the MCP tool definition for looking up a specific invoice.
func (s *InvoiceService) LookupInvoiceTool() mcp.Tool {
	return mcp.Tool{
		Name:        "lnc_lookup_invoice",
		Description: "Look up a specific invoice by its payment hash",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"payment_hash": map[string]any{
					"type":        "string",
					"description": "Payment hash of the invoice (hex encoded)",
					"pattern":     "^[0-9a-fA-F]{64}$",
				},
			},
			Required: []string{"payment_hash"},
		},
	}
}

// HandleLookupInvoice handles the lookup invoice request.
func (s *InvoiceService) HandleLookupInvoice(ctx context.Context,
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

	rhashBytes, err := hex.DecodeString(paymentHash)
	if err != nil {
		return mcp.NewToolResultError("invalid payment_hash format"), nil
	}

	// Lookup the invoice
	invoice, err := s.LightningClient.LookupInvoice(ctx, &lnrpc.PaymentHash{
		RHash: rhashBytes,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to lookup invoice: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{
		"memo": "%s",
		"payment_request": "%s",
		"r_hash": "%s",
		"value": %d,
		"value_msat": %d,
		"settled": %t,
		"creation_date": %d,
		"settle_date": %d,
		"expiry": %d,
		"cltv_expiry": %d,
		"private": %t,
		"add_index": %d,
		"settle_index": %d,
		"amt_paid_sat": %d,
		"amt_paid_msat": %d,
		"state": "%s",
		"is_keysend": %t
	}`,
		invoice.Memo,
		invoice.PaymentRequest,
		hex.EncodeToString(invoice.RHash),
		invoice.Value,
		invoice.ValueMsat,
		invoice.State == lnrpc.Invoice_SETTLED,
		invoice.CreationDate,
		invoice.SettleDate,
		invoice.Expiry,
		invoice.CltvExpiry,
		invoice.Private,
		invoice.AddIndex,
		invoice.SettleIndex,
		invoice.AmtPaidSat,
		invoice.AmtPaidMsat,
		invoice.State.String(),
		invoice.IsKeysend,
	)), nil
}
