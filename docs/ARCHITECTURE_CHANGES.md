# Architecture Changes - HTTP Server Redesign

## Overview

The SPOCP HTTP server architecture has been redesigned to provide clearer separation between monitoring and authorization concerns.

## Key Changes

### Before (Old Architecture)

- Three independent flags: `-tcp`, `-http`, `-health`
- Confusion about which port serves what
- Health endpoints duplicated across servers
- HTTP server was purely for AuthZen

### After (New Architecture)

- **HTTP Server**: Always runs, provides monitoring endpoints
- **TCP Server**: Optional via `-tcp` flag
- **AuthZen API**: Optional via `-authzen` flag on HTTP server
- Clear separation: monitoring vs authorization

## Flag Changes

| Old Flag | New Flag | Purpose |
|----------|----------|---------|
| `-tcp` (default: true) | `-tcp` (default: false) | Enable TCP server |
| `-addr :6000` | `-tcp-addr :6000` | TCP server address |
| `-http` | `-authzen` | Enable AuthZen API on HTTP |
| `-http-addr :8000` | `-http-addr :8000` | HTTP server address (monitoring + optional AuthZen) |
| `-health :8080` | *(removed)* | Monitoring now always on HTTP server |

## Deployment Modes

### 1. TCP-only + Monitoring

```bash
./spocpd -tcp -tcp-addr :6000 -http-addr :8000 -rules ./rules
```

- TCP server on `:6000` (SPOCP protocol)
- HTTP server on `:8000` (monitoring only)
- Endpoints: `/health`, `/ready`, `/stats`, `/metrics`

### 2. AuthZen-only

```bash
./spocpd -authzen -http-addr :8000 -rules ./rules
```

- HTTP server on `:8000` (monitoring + AuthZen API)
- Endpoints: `/health`, `/ready`, `/stats`, `/metrics`, `/access/v1/evaluation`

### 3. Dual-protocol

```bash
./spocpd -tcp -tcp-addr :6000 -authzen -http-addr :8000 -rules ./rules
```

- TCP server on `:6000` (SPOCP protocol)
- HTTP server on `:8000` (monitoring + AuthZen API)
- Both protocols share the same rule engine
- All endpoints available

## HTTP Endpoints

### Always Available (Monitoring)

- `GET /health` - Health check, returns `{"status":"ok"}`
- `GET /ready` - Readiness check, verifies rules are loaded
- `GET /stats` - JSON statistics (requests, rules, indexing)
- `GET /metrics` - Prometheus-style metrics

### Optional (AuthZen API)

- `POST /access/v1/evaluation` - AuthZen Authorization API (only when `-authzen` enabled)

## Code Changes

### `cmd/spocpd/main.go`

- Removed flags: `-http`, `-health`, `-addr`
- Added flags: `-tcp`, `-authzen`, `-tcp-addr`
- HTTP server always created (for monitoring)
- TCP server conditionally created (when `-tcp` enabled)
- AuthZen feature flag passed to HTTP server config
- Validation: requires at least one of `-tcp` or `-authzen`

### `pkg/httpserver/httpserver.go`

- Added `EnableAuthZen bool` to `Config` struct
- Conditional route registration for `/access/v1/evaluation`
- Monitoring endpoints always registered
- Package documentation updated to reflect new model

## Documentation Updates

All documentation has been updated to reflect the new architecture:

- ✅ **README.md**: Updated Quick Start and Features sections
- ✅ **docs/AUTHZEN.md**: Updated deployment examples and monitoring section
- ✅ **docs/adr/09-dual-protocol-server.md**: Updated with new architecture and flags
- ✅ **docs/adr/11-http-server-modes.md**: Added EnableAuthZen documentation
- ✅ **CHANGELOG.md**: Comprehensive changelog entry for architecture changes
- ✅ **Package godoc**: Updated httpserver package documentation

## Benefits

1. **Clearer Intent**: Flags clearly indicate what they enable (protocol vs API vs monitoring)
2. **Always Observable**: HTTP monitoring available in all deployment modes
3. **Reduced Confusion**: No more separate health port, everything on HTTP server
4. **Flexible**: Can run TCP-only, AuthZen-only, or both
5. **Secure**: AuthZen API only exposed when explicitly enabled

## Migration Guide

### From Old Flags to New Flags

Old command:
```bash
./spocpd -http -http-addr :8000 -rules ./rules
```

New equivalent:
```bash
./spocpd -authzen -http-addr :8000 -rules ./rules
```

---

Old command:
```bash
./spocpd -tcp -addr :6000 -http -http-addr :8000 -health :8080 -rules ./rules
```

New equivalent:
```bash
./spocpd -tcp -tcp-addr :6000 -authzen -http-addr :8000 -rules ./rules
```
(Note: Health endpoints now on HTTP server at `:8000`)

---

Old command:
```bash
./spocpd -rules ./rules
```

New equivalent:
```bash
./spocpd -tcp -tcp-addr :6000 -http-addr :8000 -rules ./rules
```
(Note: HTTP monitoring now available on `:8000`)

## Testing

All three deployment modes have been tested:

✅ AuthZen-only mode works (HTTP monitoring + AuthZen API)
✅ TCP-only mode works (HTTP monitoring, AuthZen returns 404)
✅ Dual mode works (TCP + HTTP monitoring + AuthZen API, shared engine)
✅ Health endpoints work in all modes
✅ Build successful with no errors
