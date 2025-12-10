# Dual-Protocol Server Architecture

## Decision

The SPOCP daemon supports both TCP (native SPOCP protocol) and HTTP (AuthZen API) protocols simultaneously, with independent enable/disable flags.

Command-line flags:
- `-tcp`: Enable TCP server (default: true)
- `-http`: Enable HTTP server (default: false)
- `-http-addr`: HTTP listen address (default: ":8000")

At least one protocol must be enabled.

## Rationale

**Flexibility**: Different deployment scenarios have different requirements:
- Legacy systems need TCP-only mode
- Modern microservices prefer HTTP/REST APIs
- Transition period requires both protocols

**Resource Efficiency**: When both protocols are enabled, they share a single SPOCP engine instance with synchronized access via RWMutex, reducing memory footprint and ensuring consistency.

**Backward Compatibility**: TCP protocol remains the default, ensuring existing deployments continue to work without configuration changes.

**Graceful Degradation**: Either protocol can fail independently without affecting the other. Signal handling coordinates graceful shutdown of all active protocols.

## Consequences

**Positive**:
- Supports gradual migration from TCP to HTTP
- Single binary for multiple deployment modes
- Shared engine reduces duplication
- Clear separation of concerns (pkg/server vs pkg/httpserver)

**Negative**:
- Increased complexity in main() initialization logic
- Shutdown coordination requires careful synchronization
- Two codepaths to maintain and test

**Implementation Notes**:
- Server.GetEngine() and Server.GetEngineMutex() expose engine for sharing
- HTTP server accepts optional Engine and EngineMutex in config
- Standalone HTTP mode creates its own engine from RulesDir
- shutdownComplete channel coordinates clean shutdown
