# Dual-Protocol Server Architecture

## Decision

The SPOCP daemon supports both TCP (native SPOCP protocol) and HTTP (AuthZen API + monitoring) protocols with independent enable/disable flags.

### Architecture

- **HTTP Server**: Always started, provides monitoring endpoints (`/health`, `/ready`, `/stats`, `/metrics`)
- **TCP Server**: Optional, enabled via `-tcp` flag
- **AuthZen API**: Optional, enabled via `-authzen` flag on HTTP server

Command-line flags:
- `-tcp`: Enable TCP server (default: false)
- `-tcp-addr`: TCP listen address (default: ":6000")
- `-authzen`: Enable AuthZen API on HTTP server (default: false)
- `-http-addr`: HTTP server address for monitoring and optional AuthZen (default: ":8000")

At least one protocol must be enabled (either `-tcp` or `-authzen`).

### Deployment Modes

1. **TCP-only + Monitoring**: `-tcp` (HTTP provides monitoring for TCP server)
2. **AuthZen-only**: `-authzen` (HTTP provides monitoring + AuthZen API)
3. **Dual-protocol**: `-tcp -authzen` (Both protocols share engine, HTTP provides monitoring + AuthZen API)

## Rationale

**Universal Monitoring**: HTTP server always provides operational visibility (health, stats, metrics) regardless of which authorization protocols are enabled. This simplifies monitoring across all deployment modes.

**Flexibility**: Different deployment scenarios have different requirements:
- Legacy systems need TCP-only mode
- Modern microservices prefer HTTP/REST APIs
- Transition period requires both protocols

**Resource Efficiency**: When both protocols are enabled, they share a single SPOCP engine instance with synchronized access via RWMutex, reducing memory footprint and ensuring consistency.

**Clear Separation**: The HTTP server has two distinct roles:
1. Monitoring interface (always enabled)
2. Authorization API (optional)

**Backward Compatibility**: Existing deployments can continue using TCP, with HTTP monitoring provided automatically.

**Graceful Degradation**: Either protocol can fail independently without affecting the other. Signal handling coordinates graceful shutdown of all active protocols.

## Consequences

**Positive**:
- Monitoring always available via HTTP endpoints
- Supports gradual migration from TCP to HTTP
- Single binary for multiple deployment modes
- Shared engine reduces duplication when both protocols enabled
- Clear separation of concerns (pkg/server for TCP, pkg/httpserver for HTTP/AuthZen)
- AuthZen can be enabled/disabled without affecting TCP
- Reduced attack surface when AuthZen is disabled

**Negative**:
- Increased complexity in main() initialization logic
- Shutdown coordination requires careful synchronization
- Multiple code paths to maintain and test
- HTTP server always runs even in TCP-only mode (minimal overhead)

**Implementation Notes**:
- Server.GetEngine() and Server.GetEngineMutex() expose engine for sharing
- HTTP server accepts optional Engine and EngineMutex in config
- Standalone HTTP mode creates its own engine from RulesDir
- `Config.EnableAuthZen` controls whether AuthZen endpoint is registered
- shutdownComplete channel coordinates clean shutdown
- HTTP routes conditionally registered based on EnableAuthZen flag
