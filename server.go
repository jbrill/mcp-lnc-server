// Package main provides the MCP server implementation for Lightning Network
// Connect. It exposes Model Context Protocol (MCP) tools that let AI assistants
// interact with Lightning Network nodes through Lightning Node Connect (LNC).
package main

import (
	"context"

	"github.com/jbrill/mcp-lnc-server/internal/config"
	lnccontext "github.com/jbrill/mcp-lnc-server/internal/context"
	"github.com/jbrill/mcp-lnc-server/internal/logging"
	"github.com/jbrill/mcp-lnc-server/internal/services"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server owns the MCP server instance and registered tool set.
type Server struct {
	cfg            *config.Config
	logger         *zap.Logger
	mcpServer      *server.MCPServer
	serviceManager *services.Manager
}

// NewServer creates a new MCP server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Initialize context logger.
	logging.InitContextLogger()

	// Create MCP server.
	mcpServer := server.NewMCPServer(cfg.ServerName, cfg.ServerVersion)

	// Initialize service manager for read-only operations.
	serviceManager := services.NewManager(logger)
	serviceManager.InitializeServices()

	// Register all tools with the MCP server.
	if err := serviceManager.RegisterTools(mcpServer); err != nil {
		return nil, err
	}

	return &Server{
		cfg:            cfg,
		logger:         logger,
		mcpServer:      mcpServer,
		serviceManager: serviceManager,
	}, nil
}

// Start runs the MCP server and blocks until it is stopped.
func (s *Server) Start() error {
	ctx := lnccontext.New(context.Background(), "mcp_server_start", 0)
	defer ctx.Cancel()
	logger := logging.LogWithContext(ctx)

	logger.Info("MCP Server ready - listening on stdio...",
		zap.String("server_name", s.cfg.ServerName),
		zap.String("version", s.cfg.ServerVersion))

	return server.ServeStdio(s.mcpServer)
}

// Stop gracefully stops the MCP server.
func (s *Server) Stop(ctx context.Context) error {
	reqCtx := lnccontext.Ensure(ctx, "mcp_server_stop")
	defer reqCtx.Cancel()
	logger := logging.LogWithContext(reqCtx)

	logger.Info("Stopping MCP server...")

	// Shutdown the service manager.
	if err := s.serviceManager.Shutdown(); err != nil {
		logger.Error("Error shutting down service manager",
			zap.Error(err))
		return err
	}

	logger.Info("MCP server stopped successfully",
		zap.Duration("shutdown_duration", reqCtx.Duration()))
	return nil
}
