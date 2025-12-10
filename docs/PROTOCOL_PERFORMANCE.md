# Protocol Performance Comparison

This document describes the `protocolperf` tool which compares the performance overhead of different access methods to the SPOCP authorization engine.

## Overview

The benchmark measures three access methods:

1. **Direct Engine Access** (baseline) - Library calls directly to the SPOCP engine
2. **TCP Protocol** - SPOCP binary protocol over TCP socket using the client library
3. **HTTP/AuthZen Protocol** - JSON over HTTP using the AuthZen Authorization API 1.0

All tests use the same rules and queries to ensure fair comparison. The benchmark eliminates external factors by:
- Using library calls rather than external binaries
- Running servers and clients in the same process
- Using connection pooling for HTTP
- Using the same engine instance for shared state

## Running the Benchmark

```bash
# Basic comparison (1000 rules, 10000 queries, single client)
make protocolperf

# With concurrent clients (better for throughput measurement)
make protocolperf-concurrent

# Custom parameters
go run cmd/protocolperf/main.go -rules 5000 -queries 20000 -concurrent 8
```

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-rules` | 1000 | Number of rules to generate |
| `-queries` | 10000 | Number of queries to run |
| `-concurrent` | 1 | Number of concurrent clients (for TCP/HTTP) |
| `-warmup` | 100 | Number of warmup queries |
| `-tcp-port` | 16000 | TCP server port |
| `-http-port` | 18000 | HTTP server port |
| `-skip-tcp` | false | Skip TCP benchmark |
| `-skip-http` | false | Skip HTTP benchmark |
| `-verbose` | false | Verbose output |

## Example Results

### Single Client (Latency Focus)

```
Rules: 1000, Queries: 10000, Concurrent: 1

╔═══════════════════════════════════════════════════════════════╗
║                     PERFORMANCE SUMMARY                       ║
╠═══════════════════════════════════════════════════════════════╣
║  Direct Engine   :     128,268 q/s       8µs latency  (baseline)
║  TCP (1 client)  :      22,353 q/s      45µs latency  (+474% overhead)
║  HTTP (1 client) :       7,437 q/s     134µs latency  (+1625% overhead)
╚═══════════════════════════════════════════════════════════════╝

📊 Overhead Analysis:
   TCP (1 client): +37µs per query (5.7x slower than direct)
   HTTP (1 client): +127µs per query (17.2x slower than direct)
```

### Multiple Clients (Throughput Focus)

```
Rules: 1000, Queries: 10000, Concurrent: 4

╔═══════════════════════════════════════════════════════════════╗
║                     PERFORMANCE SUMMARY                       ║
╠═══════════════════════════════════════════════════════════════╣
║  Direct Engine   :     104,378 q/s      10µs latency  (baseline)
║  TCP (4 clients) :      44,190 q/s      23µs latency  (+136% overhead)
║  HTTP (4 clients):      22,626 q/s      44µs latency  (+361% overhead)
╚═══════════════════════════════════════════════════════════════╝
```

## Performance Characteristics

### Per-Query Latency Overhead

| Protocol | Per-Query Overhead | Multiplier |
|----------|-------------------|------------|
| TCP | ~35-45µs | 5-6x baseline |
| HTTP/AuthZen | ~120-140µs | 15-20x baseline |

### Breakdown of HTTP Overhead

The HTTP/AuthZen endpoint has additional overhead compared to TCP due to:

1. **JSON serialization/deserialization** (~10-20µs)
   - Marshal request to JSON
   - Unmarshal JSON response
   
2. **HTTP protocol overhead** (~30-50µs)
   - HTTP headers parsing
   - Content-Type handling
   - Connection management
   
3. **AuthZen-to-SPOCP conversion** (~5-10µs)
   - Convert AuthZen structure to S-expression
   - Build query elements

### TCP Overhead Breakdown

1. **S-expression serialization** (~5-10µs)
   - Convert to canonical form
   
2. **Protocol framing** (~5-10µs)
   - Message encoding/decoding
   - Response parsing
   
3. **Socket I/O** (~20-30µs)
   - Network round-trip (localhost)
   - Buffer management

## When to Use Each Protocol

### Direct Engine Access
- **Use when**: Running authorization in-process (embedded)
- **Best for**: High-performance services, sidecar pattern
- **Latency**: ~5-10µs per query

### TCP Protocol
- **Use when**: Need efficient binary protocol, low latency
- **Best for**: Internal services, high-throughput systems
- **Latency**: ~40-50µs per query

### HTTP/AuthZen Protocol
- **Use when**: Standards compliance required, REST API preferred
- **Best for**: Microservices, external integrations, polyglot environments
- **Latency**: ~130-150µs per query

## Throughput vs Latency Trade-offs

- **Single client**: Direct engine has lowest latency
- **Multiple clients**: Parallel queries can increase total throughput
- **Connection pooling**: HTTP benefits from keep-alive connections
- **TCP multiplexing**: Each client maintains one connection

## Testing Methodology

1. **Rules**: Generated with realistic AuthZen structure (subject, resource, action)
2. **Queries**: 50% from existing rules (guaranteed matches) + 50% random
3. **Warmup**: 100 queries discarded before measurement
4. **Match rate**: Consistent ~50% across all protocols (validates correctness)
5. **Timing**: Wall-clock time for total query batch

## Reproducibility

The benchmark uses a fixed random seed (42) for reproducibility. Running with the same parameters should produce similar relative results, though absolute throughput depends on hardware.
