# SPOCP TCP Server and Client

This directory contains implementations of the SPOCP TCP protocol as specified in `docs/draft-hedberg-spocp-tcp-00.txt`.

## Architecture

### Protocol (pkg/protocol)
- Implements the length-value (LV) encoding format
- Message encoding/decoding
- Response handling
- S-expression parsing integration

### Server (pkg/server)
- TCP server with optional TLS support
- Dynamic rule loading from `.spoc` files
- Automatic rule reloading
- Concurrent client handling
- Graceful shutdown

### Client (pkg/client)
- TCP client with optional TLS support
- Simple API for QUERY, ADD, and RELOAD operations
- Connection management

## Quick Start

### 1. Start the Server

```bash
# Build the server
go build -o spocpd ./cmd/spocpd

# Start with a rules directory
./spocpd -rules ./examples/rules

# With TLS
./spocpd -rules ./examples/rules \
         -tls-cert server.crt \
         -tls-key server.key

# With auto-reload every 5 minutes
./spocpd -rules ./examples/rules -reload 5m
```

### 2. Use the Client

```bash
# Build the client
go build -o spocp-client ./cmd/spocp-client

# Interactive mode
./spocp-client -addr localhost:6000

# Single query
./spocp-client -query '(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:john))'

# Add a rule
./spocp-client -add '(4:http(4:page8:test.php)(6:action3:GET)(6:userid))'

# With TLS
./spocp-client -addr localhost:6000 -tls
```

## Rule Files

Rule files must have a `.spoc` extension and contain canonical S-expressions, one per line.

Example (`rules/http.spoc`):
```
(4:http(4:page10:index.html)(6:action3:GET)(6:userid))
(4:http(4:page9:admin.php)(6:action)(6:userid5:admin))
```

Comments are supported (lines starting with `#`, `//`, or `;`):
```
# HTTP access rules
(4:http(4:page10:index.html)(6:action3:GET)(6:userid))

// Admin access
(4:http(4:page9:admin.php)(6:action)(6:userid5:admin))
```

## Server Options

```
-addr string
    Address to listen on (default ":6000")
-rules string
    Directory containing .spoc rule files (required)
-tls-cert string
    Path to TLS certificate file (optional)
-tls-key string
    Path to TLS private key file (optional)
-reload duration
    Auto-reload interval (e.g., 5m, 1h) - 0 to disable (default 0)
```

## Client Options

```
-addr string
    Server address (default "localhost:6000")
-tls
    Use TLS
-insecure
    Skip TLS certificate verification
-query string
    Execute single query and exit
-add string
    Add single rule and exit
```

## Protocol Operations

### QUERY
Check if a query matches any rule in the engine.

Request:
```
70:5:QUERY60:(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))
```

Response:
- `9:3:2002:Ok` - Query matched
- `11:3:4007:Denied` - Query did not match

### ADD
Add a new rule to the engine.

Request:
```
49:3:ADD41:(4:http(4:page)(6:action3:GET)(6:userid))
```

Response:
- `9:3:2002:Ok` - Rule added successfully

### RELOAD
Reload all rules from the rules directory (custom extension).

Request:
```
9:6:RELOAD
```

Response:
- `14:3:20010:Reloaded` - Rules reloaded successfully

### LOGOUT
Close the connection gracefully.

Request:
```
8:6:LOGOUT
```

Response:
- `10:3:2033:Bye` - Connection closing

## TLS Setup

Generate self-signed certificates for testing:

```bash
# Generate private key
openssl genrsa -out server.key 2048

# Generate certificate
openssl req -new -x509 -key server.key -out server.crt -days 365 \
  -subj "/CN=localhost"
```

Start server with TLS:
```bash
./spocpd -rules ./examples/rules -tls-cert server.crt -tls-key server.key
```

Connect with client:
```bash
# With proper certificate
./spocp-client -tls

# Skip verification (testing only)
./spocp-client -tls -insecure
```

## Dynamic Rule Reloading

The server supports two modes of rule reloading:

### 1. Automatic Reloading

```bash
# Reload every 5 minutes
./spocpd -rules ./examples/rules -reload 5m
```

### 2. Manual Reloading

Send a RELOAD command from the client:
```bash
./spocp-client
> reload
✓ Server rules reloaded
```

Or programmatically:
```go
client.Reload()
```

## Programming Examples

### Server

```go
package main

import (
    "log"
    "github.com/sirosfoundation/go-spocp/pkg/server"
)

func main() {
    config := &server.Config{
        Address:  ":6000",
        RulesDir: "./rules",
    }
    
    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Fatal(srv.Serve())
}
```

### Client

```go
package main

import (
    "fmt"
    "log"
    "github.com/sirosfoundation/go-spocp/pkg/client"
)

func main() {
    config := &client.Config{
        Address: "localhost:6000",
    }
    
    c, err := client.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Query
    result, err := c.QueryString("(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:john))")
    if err != nil {
        log.Fatal(err)
    }
    
    if result {
        fmt.Println("Query matched!")
    } else {
        fmt.Println("Query denied")
    }
    
    // Add rule
    err = c.AddString("(4:http(4:page8:new.html)(6:action3:GET)(6:userid))")
    if err != nil {
        log.Fatal(err)
    }
}
```

## Performance Considerations

- The server uses tag-based indexing for efficient rule matching
- Concurrent clients are handled in separate goroutines
- Rule reloading creates a new engine and swaps atomically (no downtime)
- Connection pooling is recommended for high-throughput applications

## Security Considerations

1. **Always use TLS in production** - The protocol sends rules and queries in clear text
2. **Validate certificates** - Don't use `-insecure` in production
3. **Firewall** - Limit access to the SPOCP port
4. **Authentication** - The protocol doesn't include authentication; use TLS client certificates or a reverse proxy
5. **Rule validation** - Invalid rules are logged but don't crash the server

## Protocol Specification

For complete protocol details, see:
- `docs/draft-hedberg-spocp-tcp-00.txt` - TCP protocol specification
- `docs/draft-hedberg-spocp-sexp-00.txt` - S-expression format

## Troubleshooting

### Connection refused
- Check if server is running: `netstat -an | grep 6000`
- Verify firewall rules
- Check server logs

### TLS handshake failed
- Verify certificate and key files exist
- Check certificate validity: `openssl x509 -in server.crt -text -noout`
- Ensure client and server TLS settings match

### Rules not loading
- Check file extension is `.spoc`
- Verify file permissions
- Check server logs for parse errors
- Validate S-expression syntax

### High memory usage
- Large rulesets require more memory
- Consider splitting rules across multiple servers
- Monitor with `./spocpd -reload 0` to disable auto-reload

## Testing

Run protocol tests:
```bash
go test ./pkg/protocol
go test ./pkg/server
go test ./pkg/client
```

## See Also

- [FILE_LOADING.md](docs/FILE_LOADING.md) - File format details
- [ADAPTIVE_ENGINE.md](docs/ADAPTIVE_ENGINE.md) - Engine performance
- [OPTIMIZATION_SUMMARY.md](docs/OPTIMIZATION_SUMMARY.md) - Performance tuning
