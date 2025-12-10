# HTTP Server Operational Modes

## Decision

The HTTP server supports two operational modes and optional AuthZen API enablement:

### Operational Modes

1. **Standalone Mode**: Server creates and manages its own SPOCP engine
   - Requires: `RulesDir` (path to .spoc files)
   - Server loads rules from directory at startup
   - Engine lifecycle managed by HTTP server

2. **Shared Mode**: Server uses an external engine instance
   - Requires: `Engine` (pre-created engine) and `EngineMutex` (for synchronization)
   - Typically used when TCP and HTTP servers run together
   - Engine lifecycle managed by external component (e.g., TCP server)

### AuthZen API Control

The HTTP server always provides monitoring endpoints (`/health`, `/ready`, `/stats`, `/metrics`), but the AuthZen API endpoint (`/access/v1/evaluation`) is optional:

- `EnableAuthZen: false` (default): Only monitoring endpoints are registered
- `EnableAuthZen: true`: AuthZen API endpoint is registered in addition to monitoring endpoints

This allows the HTTP server to serve as a universal monitoring interface for SPOCP, whether running TCP-only, AuthZen-only, or both protocols.

## Rationale

**Standalone Mode** enables HTTP-only deployments:

- No need to understand TCP protocol or server package
- Simpler configuration (just point to rules directory)
- Ideal for containerized microservices
- Automatic rule loading on startup

**Shared Mode** enables dual-protocol deployments:

- Single source of truth for authorization rules
- Memory efficient (one engine, multiple protocols)
- Consistent decisions across protocols
- Synchronized access prevents race conditions

**Optional AuthZen** provides deployment flexibility:

- TCP-only deployments still get HTTP monitoring without AuthZen API
- AuthZen can be enabled/disabled independently from engine mode
- Clearer separation between monitoring and authorization API
- Reduces attack surface when AuthZen is not needed

## Consequences

**Positive**:

- Single HTTP server implementation supports both engine modes and both API modes
- Clear separation via Config struct
- Startup validation ensures exactly one engine mode is active
- `NewHTTPServer()` returns error if config is invalid
- Monitoring always available for operational visibility
- AuthZen API only exposed when explicitly enabled

**Negative**:

- More complex configuration documentation
- Multiple code paths for engine initialization and route registration
- Risk of misconfiguration (e.g., providing both RulesDir and Engine)

**Implementation Notes**:

- `Config.Engine == nil` triggers standalone mode
- `Config.Engine != nil` triggers shared mode
- `Config.EnableAuthZen` controls AuthZen endpoint registration
- Validation in `NewHTTPServer()` rejects invalid combinations
- Engine mutex defaults to server's internal mutex if not provided in shared mode
- `loadRulesFromDir()` helper handles recursive rule loading in standalone mode
- Route registration is conditional: monitoring routes always registered, AuthZen route only when enabled

## Examples

Standalone mode with AuthZen:

```go
srv, _ := httpserver.NewHTTPServer(&Config{
    Address: ":8000",
    EnableAuthZen: true,
    RulesDir: "/etc/spocp/rules",
})
```

Shared mode with TCP, monitoring only (no AuthZen):

```go
tcpSrv := server.NewServer(...)
httpSrv, _ := httpserver.NewHTTPServer(&Config{
    Address: ":8000",
    EnableAuthZen: false,  // Only monitoring endpoints
    Engine: tcpSrv.GetEngine(),
    EngineMutex: tcpSrv.GetEngineMutex(),
})
```

Shared mode with both TCP and AuthZen:

```go
tcpSrv := server.NewServer(...)
httpSrv, _ := httpserver.NewHTTPServer(&Config{
    Address: ":8000",
    EnableAuthZen: true,  // Monitoring + AuthZen API
    Engine: tcpSrv.GetEngine(),
    EngineMutex: tcpSrv.GetEngineMutex(),
})
```
