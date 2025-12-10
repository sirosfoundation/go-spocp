# SPOCP Server Operations Guide

## Overview

The SPOCP server (`spocpd`) is a production-ready authorization server with enterprise features:

- **Minimal logging by default** - Silent operation with configurable log levels
- **PID file management** - Process tracking and management
- **Health checks** - HTTP endpoints for monitoring and orchestration
- **Prometheus metrics** - Performance and operational metrics
- **Atomic rule reloading** - Zero-downtime rule updates
- **Graceful shutdown** - Clean connection termination

## Quick Start

### Basic Usage

```bash
# Minimal configuration (silent logging)
./spocpd -rules /etc/spocp/rules

# Production configuration
./spocpd \
    -addr :6000 \
    -rules /etc/spocp/rules \
    -pid /var/run/spocpd.pid \
    -health :8080 \
    -log info \
    -reload 5m
```

### With TLS

```bash
./spocpd \
    -rules /etc/spocp/rules \
    -tls-cert /etc/spocp/server.crt \
    -tls-key /etc/spocp/server.key \
    -health :8080
```

## Command-Line Options

### Required

- `-rules <dir>` - Directory containing `.spoc` rule files

### Network

- `-addr <address>` - Listen address (default: `:6000`)
  - Examples: `:6000`, `localhost:6000`, `192.168.1.10:6000`

### TLS

- `-tls-cert <file>` - Path to TLS certificate
- `-tls-key <file>` - Path to TLS private key
  - Both must be specified together

### Logging

- `-log <level>` - Log verbosity (default: `error`)
  - `silent` - No output (production default)
  - `error` - Errors only
  - `warn` - Warnings and errors
  - `info` - Informational messages
  - `debug` - Verbose debugging

### Process Management

- `-pid <file>` - PID file path (e.g., `/var/run/spocpd.pid`)
  - Automatically cleaned up on shutdown
  - Atomic write prevents race conditions

### Health & Monitoring

- `-health <address>` - Health check endpoint address (e.g., `:8080`)
  - Enables `/health`, `/ready`, `/stats`, and `/metrics` endpoints

### Rule Reloading

- `-reload <duration>` - Auto-reload interval (default: `0` - disabled)
  - Examples: `5m`, `1h`, `30s`
  - Uses atomic swap for zero-downtime updates

## Health Check Endpoints

When `-health` is specified, the following HTTP endpoints are available:

### `/health` - Liveness Probe

Always returns HTTP 200 if the server is running.

```bash
$ curl http://localhost:8080/health
{"status":"ok"}
```

**Use for:** Kubernetes liveness probes, load balancer health checks

### `/ready` - Readiness Probe

Returns HTTP 200 if rules are loaded and server is ready to accept requests.

```bash
$ curl http://localhost:8080/ready
{"status":"ready"}
```

Returns HTTP 503 if no rules are loaded:

```json
{"status":"not ready","reason":"no rules loaded"}
```

**Use for:** Kubernetes readiness probes, rolling deployments

### `/stats` - Detailed Statistics

Returns comprehensive server statistics in JSON format.

```bash
$ curl http://localhost:8080/stats | jq .
{
  "queries": {
    "total": 1234,
    "ok": 1100,
    "denied": 134
  },
  "adds": 42,
  "reloads": {
    "total": 5,
    "failed": 0,
    "last": "2025-12-10T15:32:52+01:00"
  },
  "connections": 156,
  "rules": {
    "loaded": 6,
    "total": 6,
    "by_tag": 0
  },
  "indexing": {
    "enabled": false,
    "tags": 0
  }
}
```

**Use for:** Monitoring dashboards, operational insights

### `/metrics` - Prometheus Metrics

Returns metrics in Prometheus exposition format.

```bash
$ curl http://localhost:8080/metrics
# HELP spocp_queries_total Total number of queries
# TYPE spocp_queries_total counter
spocp_queries_total 1234
# HELP spocp_queries_ok Total number of successful queries
# TYPE spocp_queries_ok counter
spocp_queries_ok 1100
# HELP spocp_queries_denied Total number of denied queries
# TYPE spocp_queries_denied counter
spocp_queries_denied 134
# HELP spocp_adds_total Total number of ADD operations
# TYPE spocp_adds_total counter
spocp_adds_total 42
# HELP spocp_reloads_total Total number of rule reloads
# TYPE spocp_reloads_total counter
spocp_reloads_total 5
# HELP spocp_reloads_failed Total number of failed reloads
# TYPE spocp_reloads_failed counter
spocp_reloads_failed 0
# HELP spocp_connections_total Total number of connections
# TYPE spocp_connections_total counter
spocp_connections_total 156
# HELP spocp_rules_loaded Current number of rules loaded
# TYPE spocp_rules_loaded gauge
spocp_rules_loaded 6
# HELP spocp_last_reload_timestamp_seconds Timestamp of last reload
# TYPE spocp_last_reload_timestamp_seconds gauge
spocp_last_reload_timestamp_seconds 1733844772
```

**Use for:** Prometheus scraping, Grafana dashboards

## Production Deployment

### Systemd Service

```bash
# Install service file
sudo cp examples/systemd/spocpd.service /etc/systemd/system/

# Create service user
sudo useradd -r -s /bin/false spocp

# Create directories
sudo mkdir -p /etc/spocp/rules /var/lib/spocp /var/log/spocp
sudo chown -R spocp:spocp /var/lib/spocp /var/log/spocp

# Copy rules
sudo cp examples/rules/*.spoc /etc/spocp/rules/
sudo chown -R spocp:spocp /etc/spocp

# Start service
sudo systemctl daemon-reload
sudo systemctl enable spocpd
sudo systemctl start spocpd

# Check status
sudo systemctl status spocpd
```

### Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o spocpd ./cmd/spocpd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/spocpd /usr/local/bin/
COPY examples/rules /etc/spocp/rules
EXPOSE 6000 8080
CMD ["spocpd", "-rules", "/etc/spocp/rules", "-health", ":8080", "-log", "info"]
```

Build and run:

```bash
docker build -t spocpd .
docker run -d \
    --name spocpd \
    -p 6000:6000 \
    -p 8080:8080 \
    -v /path/to/rules:/etc/spocp/rules:ro \
    spocpd
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spocpd
spec:
  replicas: 3
  selector:
    matchLabels:
      app: spocpd
  template:
    metadata:
      labels:
        app: spocpd
    spec:
      containers:
      - name: spocpd
        image: spocpd:latest
        args:
        - "-rules"
        - "/etc/spocp/rules"
        - "-health"
        - ":8080"
        - "-log"
        - "info"
        - "-reload"
        - "5m"
        ports:
        - name: spocp
          containerPort: 6000
        - name: health
          containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: health
          initialDelaySeconds: 3
          periodSeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        volumeMounts:
        - name: rules
          mountPath: /etc/spocp/rules
          readOnly: true
      volumes:
      - name: rules
        configMap:
          name: spocp-rules
---
apiVersion: v1
kind: Service
metadata:
  name: spocpd
spec:
  selector:
    app: spocpd
  ports:
  - name: spocp
    port: 6000
    targetPort: 6000
  - name: health
    port: 8080
    targetPort: 8080
```

## Rule Management

### Directory Structure

```text
/etc/spocp/rules/
├── http.spoc        # HTTP access rules
├── file.spoc        # File access rules
└── api.spoc         # API access rules
```

### Rule File Format

Files must end in `.spoc` and contain canonical S-expressions:

```scheme
# Comments are supported (#, //, ;)
(4:http(4:page10:index.html)(6:action3:GET)(6:userid))
(4:http(4:page9:admin.php)(6:action)(6:userid5:admin))
```

### Reloading Rules

**Manual reload via client:**

```bash
./spocp-client -addr localhost:6000
> reload
✓ Server rules reloaded
```

**Automatic reload:**

```bash
# Reload every 5 minutes
./spocpd -rules /etc/spocp/rules -reload 5m
```

**Zero-downtime guarantee:**
- New rules engine is built completely before swap
- Queries continue serving during reload
- Atomic replacement prevents partial states
- Failed reloads don't affect running engine

## Logging Levels

### Silent (Production Default)

```bash
./spocpd -rules /etc/spocp/rules -log silent
# No output
```

**Use for:** Production with external monitoring

### Error

```bash
./spocpd -rules /etc/spocp/rules -log error
[SPOCP] 2025/12/10 15:32:52 [ERROR] Failed to load rules: ...
```

**Use for:** Production with minimal logging

### Warn

```bash
./spocpd -rules /etc/spocp/rules -log warn
[SPOCP] 2025/12/10 15:32:52 [WARN] No .spoc files found in /etc/spocp/rules
[SPOCP] 2025/12/10 15:32:52 [ERROR] Connection timeout: ...
```

**Use for:** Staging environments

### Info

```bash
./spocpd -rules /etc/spocp/rules -log info
[SPOCP] 2025/12/10 15:32:52 [INFO] Loaded 6 rules from 2 files
[SPOCP] 2025/12/10 15:32:52 [INFO] Server listening on :6000 (plain TCP)
[SPOCP] 2025/12/10 15:32:52 [INFO] Health check endpoint listening on :8080
```

**Use for:** Development, troubleshooting

### Debug

```bash
./spocpd -rules /etc/spocp/rules -log debug
[SPOCP] 2025/12/10 15:32:52 [DEBUG] PID file written: /var/run/spocpd.pid (PID: 12345)
[SPOCP] 2025/12/10 15:32:52 [DEBUG] New connection from 192.168.1.10:54321
[SPOCP] 2025/12/10 15:32:52 [DEBUG] Received from 192.168.1.10:54321: QUERY [...]
```

**Use for:** Debugging, development

## Monitoring

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'spocpd'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Key Metrics

- `spocp_queries_total` - Total authorization queries
- `spocp_queries_ok` - Successful authorizations
- `spocp_queries_denied` - Denied authorizations
- `spocp_connections_total` - Client connections
- `spocp_rules_loaded` - Current rule count
- `spocp_reloads_total` - Rule reload count
- `spocp_reloads_failed` - Failed reload count

### Alerting

Example Prometheus alerts:

```yaml
groups:
- name: spocpd
  rules:
  - alert: SPOCPHighErrorRate
    expr: rate(spocp_reloads_failed[5m]) > 0
    annotations:
      summary: "SPOCP rule reloads failing"
      
  - alert: SPOCPNoRulesLoaded
    expr: spocp_rules_loaded == 0
    for: 1m
    annotations:
      summary: "SPOCP has no rules loaded"
      
  - alert: SPOCPHighDenialRate
    expr: rate(spocp_queries_denied[5m]) / rate(spocp_queries_total[5m]) > 0.5
    for: 5m
    annotations:
      summary: "More than 50% of queries denied"
```

## Operational Tasks

### Graceful Shutdown

```bash
# Send SIGTERM or SIGINT
kill -TERM $(cat /var/run/spocpd.pid)

# Or use Ctrl+C in foreground mode
```

The server will:

1. Stop accepting new connections
2. Wait for existing connections to close
3. Clean up PID file
4. Exit cleanly

### Log Rotation

When using file logging (redirect stdout):

```bash
# Start with log file
./spocpd -rules /etc/spocp/rules -log info > /var/log/spocp/spocpd.log 2>&1 &

# Logrotate configuration
cat > /etc/logrotate.d/spocpd <<EOF
/var/log/spocp/spocpd.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        /bin/kill -HUP \$(cat /var/run/spocpd.pid) 2>/dev/null || true
    endscript
}
EOF
```

### Backup and Restore

```bash
# Backup rules
tar -czf spocp-rules-$(date +%Y%m%d).tar.gz /etc/spocp/rules/

# Restore
tar -xzf spocp-rules-20251210.tar.gz -C /

# Reload
./spocp-client -addr localhost:6000
> reload
```

## Troubleshooting

### Server won't start

**Check permissions:**

```bash
ls -la /etc/spocp/rules/
```

**Verify rules syntax:**

```bash
# Test loading manually
go run ./cmd/spocpd -rules /etc/spocp/rules -log debug
```

**Check port availability:**

```bash
netstat -tuln | grep 6000
```

### High memory usage

Check rule count and complexity:

```bash
curl http://localhost:8080/stats | jq '.rules'
```

Consider using AdaptiveEngine for large rulesets (see [ADAPTIVE_ENGINE.md](ADAPTIVE_ENGINE.md))

### Slow queries

Enable debug logging to identify bottlenecks:

```bash
./spocpd -rules /etc/spocp/rules -log debug
```

Check indexing status:

```bash
curl http://localhost:8080/stats | jq '.indexing'
```

### Connection timeouts

Default read timeout is 5 minutes. For long-lived connections, clients should send periodic queries or use connection pooling.

## Performance Tuning

### Connection Limits

Adjust system limits:

```bash
# /etc/security/limits.conf
spocp soft nofile 65536
spocp hard nofile 65536
```

### Rule Organization

- Group related rules by tag for better indexing
- Use separate files for different domains
- Keep rule files under 10,000 lines each

### Auto-reload Interval

- Frequent reloads (< 1m): Higher CPU usage
- Infrequent reloads (> 30m): Longer time to detect changes
- Recommended: 5-15 minutes for most use cases

## Security Considerations

1. **Always use TLS in production**
2. **Restrict health endpoint access** (firewall or internal network only)
3. **Run as non-root user** (use systemd example)
4. **Validate rule files** before deploying
5. **Monitor failed reload attempts**
6. **Use PID file** to prevent multiple instances

## See Also

- [TCP_SERVER.md](TCP_SERVER.md) - Protocol specification
- [FILE_LOADING.md](FILE_LOADING.md) - Rule file formats
- [ADAPTIVE_ENGINE.md](ADAPTIVE_ENGINE.md) - Performance optimization
- [OPTIMIZATION_SUMMARY.md](OPTIMIZATION_SUMMARY.md) - Tuning guide
