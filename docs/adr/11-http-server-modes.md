# HTTP Server Operational Modes

## Decision

The HTTP server supports two operational modes, configured via the `Config` struct:

1. **Standalone Mode**: Server creates and manages its own SPOCP engine
   - Requires: `RulesDir` (path to .spoc files)
   - Server loads rules from directory at startup
   - Engine lifecycle managed by HTTP server

2. **Shared Mode**: Server uses an external engine instance
   - Requires: `Engine` (pre-created engine) and `EngineMutex` (for synchronization)
   - Typically used when TCP and HTTP servers run together
   - Engine lifecycle managed by external component (e.g., TCP server)

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

## Consequences

**Positive**:

- Single HTTP server implementation supports both use cases
- Clear separation via Config struct
- Startup validation ensures exactly one mode is active
- `NewHTTPServer()` returns error if config is invalid

**Negative**:

- More complex configuration documentation
- Two code paths for engine initialization
- Risk of misconfiguration (e.g., providing both RulesDir and Engine)

**Implementation Notes**:

- `Config.Engine == nil` triggers standalone mode
- `Config.Engine != nil` triggers shared mode
- Validation in `NewHTTPServer()` rejects invalid combinations
- Engine mutex defaults to server's internal mutex if not provided in shared mode
- `loadRulesFromDir()` helper handles recursive rule loading in standalone mode

## Examples

Standalone:

```go
srv, _ := httpserver.NewHTTPServer(&Config{
    Address: ":8000",
    RulesDir: "/etc/spocp/rules",
})
```

Shared:

```go
tcpSrv := server.NewServer(...)
httpSrv, _ := httpserver.NewHTTPServer(&Config{
    Address: ":8000",
    Engine: tcpSrv.GetEngine(),
    EngineMutex: tcpSrv.GetEngineMutex(),
})
```
