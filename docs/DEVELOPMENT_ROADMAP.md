# MCP-LNC Development Roadmap & Contributor Guide

## Overview

The MCP-LNC Server bridges AI assistants with the Lightning Network through Lightning Node Connect (LNC). While the core architecture is solid, several key areas need development - particularly around version compatibility, session management, and tool optimization. **This is a new area where Lightning Network and LNC expertise would be invaluable.**

## Current Status & Architecture

Built as a Go service that translates MCP tool calls into Lightning Network operations via LNC:
- **MCP Layer**: Handles Claude Desktop stdio communication  
- **Service Layer**: Domain-specific Lightning operations (payments, channels, etc.)
- **LNC Layer**: Secure WebSocket tunneling to Lightning nodes
- **Version Layer**: Compatibility management across LND versions

**Repository**: Currently at `github.com/jbrill/mcp-lnc-server` - may move to Lightning Labs org when ready.

## Priority Development Areas

### 1. 🔥 Version Compatibility System (HIGH PRIORITY)

**Current State**: Basic version detection exists but needs significant work.

**TODOs**:
- [ ] **Dynamic gRPC introspection** - Would require LND-side changes but enables true dynamic compatibility
- [ ] **Tool-to-version range mapping** - Each MCP tool should declare supported LND version ranges
- [ ] **Graceful fallback chains** - When newer APIs unavailable, fall back through older methods
- [ ] **Field availability detection** - Runtime detection of deprecated/new protobuf fields
- [ ] **Version-specific testing matrix** - Automated testing across LND v0.16+ versions

**Why LNC Knowledge Helps**: Understanding LNC evolution across LND versions, knowing which LNC features map to which LND capabilities, experience with Lightning Labs' deprecation patterns.

```go
// Example: Tool version mapping needed
type ToolVersionSupport struct {
    ToolName: "lnc_open_channel"
    MinVersion: "v0.16.0"  
    MaxVersion: "" // latest
    DeprecatedIn: ""
    Features: map[string]string{
        "taproot_channels": "v0.17.0+",
        "zero_conf": "v0.16.0+", 
    }
}
```

### 2. 🔥 Environment Variable Session Management (HIGH PRIORITY)

**Current State**: Basic env var support exists but session handling is primitive.

**TODOs**:
- [ ] **Persistent session tokens** - Cache LNC auth beyond process lifetime
- [ ] **Session refresh logic** - Automatic renewal of expiring LNC sessions
- [ ] **Multi-session support** - Handle multiple Lightning nodes simultaneously  
- [ ] **Secure credential storage** - OS keychain integration for pairing phrases
- [ ] **Session health monitoring** - Detect and recover from stale connections

**Why LNC Knowledge Helps**: Understanding LNC session lifecycle, auth token management, WebSocket connection health patterns, mailbox server behavior.

```bash
# Example: Advanced session management needed
export LNC_SESSION_CACHE_DIR="~/.mcp-lnc/sessions"
export LNC_AUTO_REFRESH="true"  
export LNC_HEALTH_CHECK_INTERVAL="30s"
export LNC_MAX_SESSION_AGE="24h"
```

### 3. 🔥 Tool Exposure & Context Optimization (HIGH PRIORITY)

**Current State**: All 21+ tools always exposed, which pollutes Claude's context.

**TODOs**:
- [ ] **Selective tool exposure** - CLI flags/config to enable specific tool subsets
- [ ] **Tool categories** - Group by functionality (payments, channels, onchain, etc.)
- [ ] **Progressive disclosure** - Start with basic tools, expand as needed
- [ ] **Context-aware suggestions** - Recommend relevant tools based on node state
- [ ] **Custom tool bundles** - Predefined tool sets for specific use cases

**Implementation needed**:
```go
type ToolExposureConfig struct {
    EnabledCategories []string // ["payments", "channels"]
    CustomTools      []string // ["lnc_pay_invoice", "lnc_get_balance"] 
    ProgressiveMode  bool     // Start minimal, expand on demand
}
```

## Areas Seeking Lightning Network Expertise

### 🚨 Critical: LNC Protocol Deep Knowledge Needed

**Where Viktor/Boris or LNC experts could contribute:**

1. **WebSocket Connection Patterns**
   - Optimal reconnection strategies for LNC WebSocket tunnels
   - Handling mailbox server failover and load balancing
   - Connection pooling and multiplexing best practices

2. **Authentication Flow Optimization** 
   - Session token caching and refresh mechanisms
   - Pairing phrase security and storage patterns
   - Multi-node authentication handling

3. **LNC Version Compatibility Matrix**
   - Which LNC versions work with which LND versions
   - Feature availability mapping across versions  
   - Migration paths for deprecated LNC APIs

4. **Performance Optimization**
   - Efficient gRPC streaming for large operations
   - Batching strategies for bulk operations
   - Memory management for long-running connections

### 🎯 Specific Contribution Opportunities

**For someone with LNC knowledge to lead:**

1. **`internal/lnc/` package** - Dedicated LNC management layer
2. **Connection health monitoring** - Proactive connection recovery
3. **Advanced authentication** - Token refresh, multi-session support
4. **Performance profiling** - Identify LNC bottlenecks and optimizations

## Technical Architecture TODOs

### Version Compatibility Layer
```go
// Needs implementation
type VersionCompatibility interface {
    DetectCapabilities(conn *grpc.ClientConn) (*Capabilities, error)
    GetCompatibleMethod(operation string, version string) (MethodSpec, error)
    HasFeature(feature string) bool
}
```

### Session Management Layer
```go
// Needs design
type SessionManager interface {
    CreateSession(pairingPhrase, password string) (*Session, error)
    RefreshSession(sessionID string) error  
    GetActiveSession(nodeID string) (*Session, error)
    CleanupStaleSessions() error
}
```

### Tool Registry System
```go
// Needs implementation
type ToolRegistry interface {
    RegisterTool(tool MCPTool, config ToolConfig) error
    GetEnabledTools(profile string) []MCPTool
    UpdateToolAvailability(nodeCapabilities *Capabilities) error
}
```

## Development Environment Setup

### For LNC Development:
```bash
# Regtest environment with LNC
export LNC_MAILBOX_SERVER="localhost:11110" 
export LNC_DEV_MODE="true"
export LNC_INSECURE="true"
export LND_VERSION="v0.18.0-beta"

# Enable debug logging
export LOG_LEVEL="debug"
export LNC_TRACE_ENABLED="true"
```

### Testing Requirements:
- **Multi-version testing**: LND v0.16, v0.17, v0.18 compatibility
- **Connection resilience**: Network interruption recovery
- **Session persistence**: Process restart session recovery
- **Tool subset validation**: Verify selective exposure works

## Contribution Areas by Expertise

### 🧠 **Perfect for LNC Experts (Viktor/Boris)**:
- LNC connection lifecycle management
- Authentication and session optimization  
- Version compatibility edge cases
- WebSocket connection health patterns
- Performance optimization for LNC tunnels

### 🔧 **Good for Go/gRPC Developers**:
- Tool registration and exposure system
- Version detection and capability mapping
- Error handling and recovery patterns
- Testing framework and CI/CD

### 🎯 **Good for MCP/AI Integration**:
- Tool categorization and progressive disclosure
- Context optimization for AI assistants
- User experience and error messaging
- Documentation and examples

## Next Steps

1. **Immediate (This Week)**:
   - Define tool exposure configuration format
   - Basic session caching implementation
   - Version compatibility test matrix

2. **Short Term (2-4 weeks)**:
   - LNC connection health monitoring
   - Advanced session management
   - Selective tool exposure implementation

3. **Medium Term (1-2 months)**:
   - Full version compatibility system
   - Performance optimization
   - Multi-node support

## Getting Involved

**For LNC experts looking to contribute:**
1. Review current LNC usage patterns in `tools/connection.go`
2. Identify areas where deeper LNC knowledge would help
3. Consider leading the session management or connection health efforts
4. Help define the version compatibility requirements

**This represents a new area where Lightning Network expertise can directly enable AI-Lightning integration at scale.**

## Questions for LNC Experts

1. What's the optimal strategy for LNC session persistence across process restarts?
2. How should we handle LNC version compatibility as LND evolves?
3. What are the performance characteristics we should optimize for?
4. Which LNC features should we prioritize for AI assistant use cases?

**Contact**: Discuss with team about Viktor's bandwidth and interest in leading LNC-specific development areas.