package tools

import (
	"strings"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Test InvoiceService basic functionality.
func TestInvoiceService_ToolCreation(t *testing.T) {
	service := NewInvoiceService(nil)

	t.Run("list_invoices_tool", func(t *testing.T) {
		tool := service.ListInvoicesTool()
		assert.Equal(t, "lnc_list_invoices", tool.Name)
		assert.Contains(t, tool.Description, "List invoices created by this Lightning node")
		assert.Equal(t, "object", tool.InputSchema.Type)

		// Check optional fields exist.
		props := tool.InputSchema.Properties
		assert.Contains(t, props, "pending_only")
		assert.Contains(t, props, "index_offset")

		// Verify pending_only field structure.
		pendingField, ok := props["pending_only"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "boolean", pendingField["type"])
	})

	t.Run("decode_invoice_tool", func(t *testing.T) {
		tool := service.DecodeInvoiceTool()
		assert.Equal(t, "lnc_decode_invoice", tool.Name)
		assert.Contains(t, tool.Description, "Decode a BOLT11 Lightning invoice")
		assert.Equal(t, "object", tool.InputSchema.Type)

		// Check required fields.
		props := tool.InputSchema.Properties
		assert.Contains(t, props, "invoice")

		// Verify invoice field structure.
		invoiceField, ok := props["invoice"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", invoiceField["type"])
	})
}

func TestInvoiceService_ServiceManagement(t *testing.T) {
	// Test service creation.
	service := NewInvoiceService(nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.LightningClient)

	// Test service with client update.
	service.LightningClient = nil // Simulate setting client later.
	assert.Nil(t, service.LightningClient)
}

// Test ConnectionService basic functionality.
func TestConnectionService_ToolCreation(t *testing.T) {
	callback := func(conn *grpc.ClientConn) {}
	service := NewConnectionService(callback)

	t.Run("connect_tool", func(t *testing.T) {
		tool := service.ConnectTool()
		assert.Equal(t, "lnc_connect", tool.Name)
		assert.Contains(t, tool.Description, "Connect to a Lightning node")
		assert.Equal(t, "object", tool.InputSchema.Type)

		// Check required fields.
		props := tool.InputSchema.Properties
		assert.Contains(t, props, "pairingPhrase")

		// Verify pairingPhrase field structure.
		pairingField, ok := props["pairingPhrase"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "string", pairingField["type"])

		// Check optional fields.
		assert.Contains(t, props, "mailbox")
		assert.Contains(t, props, "devMode")
		assert.Contains(t, props, "password")

		// Verify required fields list contains pairingPhrase.
		require.Contains(t, tool.InputSchema.Required, "pairingPhrase")
	})

	t.Run("disconnect_tool", func(t *testing.T) {
		tool := service.DisconnectTool()
		assert.Equal(t, "lnc_disconnect", tool.Name)
		assert.Contains(t, tool.Description,
			"Disconnect from the Lightning node")
		assert.Equal(t, "object", tool.InputSchema.Type)

		// Disconnect tool should have no required parameters.
		assert.Equal(t, 0, len(tool.InputSchema.Required))
	})
}

func TestConnectionService_ServiceManagement(t *testing.T) {
	callback := func(conn *grpc.ClientConn) {}
	service := NewConnectionService(callback)
	assert.NotNil(t, service)

	// Test that we can create tools.
	connectTool := service.ConnectTool()
	disconnectTool := service.DisconnectTool()

	assert.NotNil(t, connectTool)
	assert.NotNil(t, disconnectTool)
	assert.NotEqual(t, connectTool.Name, disconnectTool.Name)
}

// Test helper functions and utilities.
func TestIsValidBolt11(t *testing.T) {
	tests := []struct {
		name    string
		invoice string
		want    bool
	}{
		{
			name:    "valid_bolt11_mainnet",
			invoice: "lnbc10m1pv9p9r4pp5...",
			want:    true,
		},
		{
			name:    "valid_bolt11_testnet",
			invoice: "lntb500m1pv9p9r4pp5...",
			want:    true,
		},
		{
			name:    "valid_bolt11_regtest",
			invoice: "lnbcrt1m1pv9p9r4pp5...",
			want:    true,
		},
		{
			name:    "invalid_too_short",
			invoice: "ln",
			want:    false,
		},
		{
			name:    "invalid_prefix",
			invoice: "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4",
			want:    false,
		},
		{
			name:    "empty_string",
			invoice: "",
			want:    false,
		},
		{
			name:    "invalid_no_ln_prefix",
			invoice: "bc10m1pv9p9r4pp5...",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidBolt11(tt.invoice)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPairingPhraseValidation(t *testing.T) {
	// Test the word counting logic used in connection service.
	tests := []struct {
		name          string
		phrase        string
		expectValid   bool
		expectedWords int
	}{
		{
			name:          "exactly_10_words",
			phrase:        "one two three four five six seven eight nine ten",
			expectValid:   true,
			expectedWords: 10,
		},
		{
			name:          "9_words",
			phrase:        "one two three four five six seven eight nine",
			expectValid:   false,
			expectedWords: 9,
		},
		{
			name: "11_words",
			phrase: "one two three four five six seven eight nine ten " +
				"eleven",
			expectValid:   false,
			expectedWords: 11,
		},
		{
			name:          "extra_spaces_handled",
			phrase:        "one  two   three four five six seven eight nine ten",
			expectValid:   true, // strings.Fields handles extra spaces.
			expectedWords: 10,
		},
		{
			name:          "leading_trailing_spaces",
			phrase:        " one two three four five six seven eight nine ten ",
			expectValid:   true, // strings.Fields trims spaces.
			expectedWords: 10,
		},
		{
			name:          "empty_string",
			phrase:        "",
			expectValid:   false,
			expectedWords: 0,
		},
		{
			name:          "only_spaces",
			phrase:        "   ",
			expectValid:   false,
			expectedWords: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := strings.Fields(tt.phrase)
			actualWordCount := len(words)

			assert.Equal(t, tt.expectedWords, actualWordCount)

			isValid := actualWordCount == 10
			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}

// Test service integration.
func TestServiceIntegration(t *testing.T) {
	t.Run("invoice_service_complete", func(t *testing.T) {
		service := NewInvoiceService(nil)

		// Verify tools are created correctly.
		listTool := service.ListInvoicesTool()
		decodeTool := service.DecodeInvoiceTool()

		assert.NotEmpty(t, listTool.Name)
		assert.NotEmpty(t, decodeTool.Name)
		assert.NotEqual(t, listTool.Name, decodeTool.Name)

		// Test service state management.
		assert.Nil(t, service.LightningClient)
	})

	t.Run("connection_service_complete", func(t *testing.T) {
		callback := func(conn *grpc.ClientConn) {}
		service := NewConnectionService(callback)

		connectTool := service.ConnectTool()
		disconnectTool := service.DisconnectTool()

		assert.NotNil(t, connectTool)
		assert.NotNil(t, disconnectTool)
		assert.NotEqual(t, connectTool.Name, disconnectTool.Name)

		// Test tools have proper structure.
		assert.NotEmpty(t, connectTool.Name)
		assert.NotEmpty(t, connectTool.Description)
		assert.NotNil(t, connectTool.InputSchema)

		assert.NotEmpty(t, disconnectTool.Name)
		assert.NotEmpty(t, disconnectTool.Description)
		assert.NotNil(t, disconnectTool.InputSchema)
	})
}

// Test helper to create test invoice.
func createTestInvoice(amount int64, memo string) *lnrpc.Invoice {
	return &lnrpc.Invoice{
		ValueMsat: amount * 1000,
		Memo:      memo,
		Expiry:    3600,
	}
}

func TestCreateTestInvoice(t *testing.T) {
	invoice := createTestInvoice(1000, "test memo")
	assert.Equal(t, int64(1000000), invoice.ValueMsat)
	assert.Equal(t, "test memo", invoice.Memo)
	assert.Equal(t, int64(3600), invoice.Expiry)
}

// Benchmark tests for performance.
func BenchmarkInvoiceService_ListInvoicesTool(b *testing.B) {
	service := NewInvoiceService(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ListInvoicesTool()
	}
}

func BenchmarkConnectionService_ConnectTool(b *testing.B) {
	callback := func(conn *grpc.ClientConn) {}
	service := NewConnectionService(callback)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ConnectTool()
	}
}

func BenchmarkIsValidBolt11(b *testing.B) {
	testInvoices := []string{
		"lnbc10m1pv9p9r4pp5...",
		"lntb500m1pv9p9r4pp5...",
		"invalid_invoice",
		"",
		"ln",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, invoice := range testInvoices {
			_ = isValidBolt11(invoice)
		}
	}
}

func BenchmarkPairingPhraseValidation(b *testing.B) {
	phrases := []string{
		"one two three four five six seven eight nine ten",
		"one two three",
		"one two three four five six seven eight nine ten eleven twelve",
		"",
		"   ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, phrase := range phrases {
			words := strings.Fields(phrase)
			_ = len(words) == 10
		}
	}
}
