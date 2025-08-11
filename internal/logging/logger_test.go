package logging

import (
	"os"
	"testing"

	"github.com/jbrill/mcp-lnc-server/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Test InitLogger with development mode.
func TestInitLogger_Development(t *testing.T) {
	err := InitLogger(true)
	require.NoError(t, err)
	assert.NotNil(t, Logger)
	
	// Clean up
	Logger = nil
}

// Test InitLogger with production mode.
func TestInitLogger_Production(t *testing.T) {
	err := InitLogger(false)
	require.NoError(t, err)
	assert.NotNil(t, Logger)
	
	// Clean up
	Logger = nil
}

// Test InitLogger with custom log level from environment.
func TestInitLogger_CustomLogLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	
	err := InitLogger(false)
	require.NoError(t, err)
	assert.NotNil(t, Logger)
	
	// Verify log level is set (we can't directly check the level easily,
	// but we can verify the logger was created without error)
	assert.True(t, Logger.Core().Enabled(zapcore.WarnLevel))
	
	// Clean up
	Logger = nil
}

// Test InitLogger with invalid log level.
func TestInitLogger_InvalidLogLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "invalid")
	defer os.Unsetenv("LOG_LEVEL")
	
	err := InitLogger(false)
	require.NoError(t, err)
	assert.NotNil(t, Logger)
	
	// Should fall back to default level
	assert.NotNil(t, Logger)
	
	// Clean up
	Logger = nil
}

// Test NewLogger creates interface wrapper.
func TestNewLogger(t *testing.T) {
	zapLogger := zap.NewNop()
	interfaceLogger := NewLogger(zapLogger)
	
	assert.NotNil(t, interfaceLogger)
	
	// Verify it implements the interface
	var _ interfaces.Logger = interfaceLogger
}

// Test zapLogger methods.
func TestZapLogger_Methods(t *testing.T) {
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)
	
	// These should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	
	// Test With method
	childLogger := logger.With(zap.String("key", "value"))
	assert.NotNil(t, childLogger)
	// Child logger should be a different instance
	assert.IsType(t, logger, childLogger)
	assert.NotSame(t, logger, childLogger)
	
	// Child logger should also implement the interface
	var _ interfaces.Logger = childLogger
}

// Test zapLogger with fields.
func TestZapLogger_WithFields(t *testing.T) {
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)
	
	// Test logging with fields
	logger.Debug("debug message", zap.String("field", "value"))
	logger.Info("info message", zap.Int("count", 42))
	logger.Warn("warn message", zap.Bool("flag", true))
	logger.Error("error message", zap.Duration("duration", 0))
}

// Test Sync function.
func TestSync(t *testing.T) {
	// Test with nil logger
	Logger = nil
	Sync() // Should not panic
	
	// Test with real logger
	err := InitLogger(true)
	require.NoError(t, err)
	
	Sync() // Should not panic
	
	// Clean up
	Logger = nil
}

// Test logger configuration consistency.
func TestLoggerConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		development bool
	}{
		{
			name:        "development_mode",
			development: true,
		},
		{
			name:        "production_mode",
			development: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.development)
			require.NoError(t, err)
			assert.NotNil(t, Logger)
			
			// Verify global logger was set
			assert.Equal(t, Logger, zap.L())
			
			// Clean up
			Logger = nil
		})
	}
}

// Test logger interface compliance.
func TestLoggerInterfaceCompliance(t *testing.T) {
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)
	
	// Test that it properly implements all interface methods
	assert.Implements(t, (*interfaces.Logger)(nil), logger)
}

// Test environment variable handling.
func TestEnvironmentVariableHandling(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		valid    bool
	}{
		{
			name:     "debug_level",
			envValue: "debug",
			valid:    true,
		},
		{
			name:     "info_level", 
			envValue: "info",
			valid:    true,
		},
		{
			name:     "warn_level",
			envValue: "warn",
			valid:    true,
		},
		{
			name:     "error_level",
			envValue: "error",
			valid:    true,
		},
		{
			name:     "invalid_level",
			envValue: "invalid",
			valid:    false,
		},
		{
			name:     "empty_level",
			envValue: "",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("LOG_LEVEL", tt.envValue)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}
			defer os.Unsetenv("LOG_LEVEL")
			
			err := InitLogger(false)
			require.NoError(t, err)
			assert.NotNil(t, Logger)
			
			// Clean up
			Logger = nil
		})
	}
}

// Benchmark logger creation.
func BenchmarkInitLogger(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InitLogger(false)
		Logger = nil
	}
}

// Benchmark NewLogger wrapper creation.
func BenchmarkNewLogger(b *testing.B) {
	zapLogger := zap.NewNop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewLogger(zapLogger)
	}
}

// Benchmark logging operations.
func BenchmarkLogging(b *testing.B) {
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", zap.Int("iteration", i))
	}
}