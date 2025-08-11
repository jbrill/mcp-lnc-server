// Package main implements the MCP LNC server daemon.
//
// The MCP LNC server provides a Model Context Protocol (MCP) interface to.
// Lightning Network Daemon (LND) nodes via Lightning Node Connect (LNC).
// This allows AI assistants like Claude to interact with Lightning Network.
// Nodes securely through WebSocket connections.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	lnccontext "github.com/jbrill/mcp-lnc-server/internal/context"
	"github.com/jbrill/mcp-lnc-server/internal/config"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"go.uber.org/zap"
)

const (
	// DefaultConfigFile is the default config file name.
	defaultConfigFile = "mcp-lnc-server.conf"

	// DefaultDataDir is the default directory for data files.
	defaultDataDir = "~/.mcp-lnc-server"
)

// Daemon represents the main daemon instance.
type Daemon struct {
	cfg    *config.Config
	logger *zap.Logger
	server *Server

	// Quit is used to signal shutdown.
	quit chan struct{}

	// ShutdownComplete is closed when all shutdown operations are complete.
	shutdownComplete chan struct{}
}

// NewDaemon creates a new daemon instance with the given configuration.
func NewDaemon(cfg *config.Config, logger *zap.Logger) (*Daemon, error) {
	server, err := NewServer(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &Daemon{
		cfg:              cfg,
		logger:           logger,
		server:           server,
		quit:             make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}, nil
}

// Start starts the daemon and blocks until it receives a shutdown signal.
func (d *Daemon) Start() error {
	// Create context for daemon startup
	ctx := lnccontext.New(context.Background(), "daemon_start", 0)
	logger := logging.LogWithContext(ctx)
	
	logger.Info("Starting MCP LNC Server daemon",
		zap.String("version", d.cfg.ServerVersion),
		zap.Bool("development", d.cfg.Development),
	)

	// Start the server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := d.server.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start shutdown handler
	go d.shutdownHandler()

	// Wait for either a shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", 
			zap.String("signal", sig.String()),
			zap.Duration("uptime", ctx.Duration()))
		close(d.quit)

	case err := <-serverErrChan:
		if err != nil && err != context.Canceled {
			logger.Error("Server error", 
				zap.Error(err),
				zap.Duration("uptime", ctx.Duration()))
			close(d.quit)
			return err
		}

	case <-d.quit:
		// Shutdown was triggered internally
	}

	// Wait for shutdown to complete
	<-d.shutdownComplete
	logger.Info("MCP LNC Server daemon shutdown complete",
		zap.Duration("total_uptime", ctx.Duration()))

	return nil
}

// Stop triggers a graceful shutdown of the daemon.
func (d *Daemon) Stop() {
	select {
	case <-d.quit:
		// Already shutting down
		return
	default:
	}

	ctx := lnccontext.New(context.Background(), "daemon_stop", 
		5*time.Second)
	logger := logging.LogWithContext(ctx)
	logger.Info("Initiating daemon shutdown...")
	close(d.quit)
}

// ShutdownHandler handles the graceful shutdown process.
func (d *Daemon) shutdownHandler() {
	<-d.quit

	// Create context for shutdown with timeout
	shutdownCtx := lnccontext.New(
		context.Background(), 
		"daemon_shutdown", 
		d.cfg.ShutdownTimeout,
	)
	logger := logging.LogWithContext(shutdownCtx)
	
	logger.Info("Beginning graceful shutdown...",
		zap.Duration("timeout", d.cfg.ShutdownTimeout))

	// Stop the server
	if err := d.server.Stop(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", 
			zap.Error(err),
			zap.Duration("shutdown_duration", shutdownCtx.Duration()))
	} else {
		logger.Info("Server shutdown completed successfully",
			zap.Duration("shutdown_duration", shutdownCtx.Duration()))
	}

	// Signal shutdown complete
	close(d.shutdownComplete)
}

// Main is the entry point for the MCP LNC server daemon.
func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logging
	if err := logging.InitLogger(cfg.Development); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logger := logging.Logger

	// Create and start the daemon
	daemon, err := NewDaemon(cfg, logger)
	if err != nil {
		logger.Error("Failed to create daemon", zap.Error(err))
		os.Exit(1)
	}

	if err := daemon.Start(); err != nil {
		logger.Error("Daemon startup failed", zap.Error(err))
		os.Exit(1)
	}
}
