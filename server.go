// Package main provides the MCP server implementation for Lightning Network
// Connect.
//
// This package implements the Model Context Protocol (MCP) server that allows
// AI assistants to interact with Lightning Network nodes through LNC.
package main

import (
	"context"

	"github.com/jbrill/mcp-lnc-server/internal/config"
	"github.com/jbrill/mcp-lnc-server/internal/services"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server represents the MCP server instance.
type Server struct {
	cfg            *config.Config
	logger         *zap.Logger
	mcpServer      *server.MCPServer
	serviceManager *services.Manager
}

// NewServer creates a new MCP server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Create MCP server
	mcpServer := server.NewMCPServer(cfg.ServerName, cfg.ServerVersion)

	// Initialize service manager
	serviceManager := services.NewManager(logger)
	serviceManager.InitializeServices()

	// Register all tools with the MCP server
	serviceManager.RegisterTools(mcpServer)

	return &Server{
		cfg:            cfg,
		logger:         logger,
		mcpServer:      mcpServer,
		serviceManager: serviceManager,
	}, nil
}

// Start starts the MCP server and blocks until it's stopped.
func (s *Server) Start() error {
	s.logger.Info("MCP Server ready - listening on stdio...")
	return server.ServeStdio(s.mcpServer)
}

// Stop gracefully stops the MCP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping MCP server...")

	// Shutdown the service manager
	s.serviceManager.Shutdown()

	s.logger.Info("MCP server stopped successfully")
	return nil
}
