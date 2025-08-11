// Package main implements the MCP LNC server daemon.
//
// The MCP LNC server provides a Model Context Protocol (MCP) interface to
// Lightning Network Daemon (LND) nodes via Lightning Node Connect (LNC).
// This allows AI assistants like Claude to interact with Lightning Network
// nodes securely through WebSocket connections.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jbrill/mcp-lnc-server/internal/config"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"go.uber.org/zap"
)

const (
	// defaultConfigFile is the default config file name.
	defaultConfigFile = "mcp-lnc-server.conf"

	// defaultDataDir is the default directory for data files.
	defaultDataDir = "~/.mcp-lnc-server"
)

// Daemon represents the main daemon instance.
type Daemon struct {
	cfg    *config.Config
	logger *zap.Logger
	server *Server

	// quit is used to signal shutdown.
	quit chan struct{}

	// shutdownComplete is closed when all shutdown operations are complete.
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
	d.logger.Info("Starting MCP LNC Server daemon",
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
		d.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		close(d.quit)

	case err := <-serverErrChan:
		if err != nil && err != context.Canceled {
			d.logger.Error("Server error", zap.Error(err))
			close(d.quit)
			return err
		}

	case <-d.quit:
		// Shutdown was triggered internally
	}

	// Wait for shutdown to complete
	<-d.shutdownComplete
	d.logger.Info("MCP LNC Server daemon shutdown complete")

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

	d.logger.Info("Initiating daemon shutdown...")
	close(d.quit)
}

// shutdownHandler handles the graceful shutdown process.
func (d *Daemon) shutdownHandler() {
	<-d.quit

	d.logger.Info("Beginning graceful shutdown...")

	// Create a context with timeout for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		d.cfg.ShutdownTimeout,
	)
	defer cancel()

	// Stop the server
	if err := d.server.Stop(shutdownCtx); err != nil {
		d.logger.Error("Error during server shutdown", zap.Error(err))
	}

	// Signal shutdown complete
	close(d.shutdownComplete)
}

// main is the entry point for the MCP LNC server daemon.
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
