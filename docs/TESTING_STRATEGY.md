# Testing Strategy for MCP LNC Server

## Overview

This document outlines the testing strategy for the MCP LNC Server, focusing on dockerized testing environments for consistent testing across different environments.

## Configuration for Claude Desktop

To run the MCP LNC server in Docker, modify your Claude Desktop MCP server configuration file (`claude_desktop_config.json`) to include the following:

```json
{
  "mcpServers": {
    "lnc-mcp-server": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "--network",
        "host",
        "--env",
        "LNC_MAILBOX_SERVER",
        "--env",
        "LNC_DEV_MODE", 
        "--env",
        "LNC_INSECURE",
        "mcp-lnc-server"
      ]
    }
  }
}
```

This configuration runs the MCP LNC server in a Docker container with proper network access and environment variable support for Lightning Network Connect configuration.

## Current Test Coverage

### Unit Test Coverage (by Package)
- **internal/config**: 88.2% - Configuration loading and environment variable handling
- **internal/context**: 95.6% - Request context management and tracing  
- **internal/errors**: 100% - Error handling and wrapping
- **internal/services**: 72.7% - Service management and tool registration
- **tools**: 1.7% - Lightning Network tool implementations (needs improvement)

## Test Categories

### ✅ **Configuration Testing**
- Environment variable loading and validation
- Development vs production mode switching
- Timeout and connection parameter validation
- Boolean flag parsing with multiple formats

### ✅ **Context Management Testing**
- Request tracing and context propagation
- Timeout and cancellation handling
- Concurrent access patterns
- Context value extraction and validation

### ✅ **Error Handling Testing**  
- Error code definitions and uniqueness
- Error wrapping and unwrapping chains
- User-friendly error message formatting
- Standard error interface compliance

### ✅ **Service Integration Testing**
- Tool registration and validation
- Service lifecycle management
- Read-only mode enforcement
- Tool name uniqueness verification

### ⚠️ **Basic Tool Testing**
- Tool creation and schema validation
- Parameter validation logic
- BOLT11 invoice format validation
- Pairing phrase format validation