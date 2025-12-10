# AuthZen HTTP Endpoint for SPOCP

The SPOCP server now supports the [AuthZen Authorization API 1.0](https://openid.net/specs/authorization-api-1_0-01.html) specification via HTTP, in addition to the native SPOCP TCP protocol.

## Features

- **Standards-compliant**: Implements AuthZen Authorization API 1.0 draft specification
- **Flexible deployment**: Run TCP-only with monitoring, AuthZen-only, or both simultaneously
- **Unified monitoring**: HTTP server always provides health/ready/stats/metrics endpoints
- **Optional AuthZen**: AuthZen API can be enabled independently via `-authzen` flag
- **Shared engine**: Both TCP and HTTP protocols can share the same rule engine
- **JSON-based**: RESTful HTTP API with JSON request/response format
- **Automatic mapping**: AuthZen requests are automatically translated to SPOCP S-expressions

## Endpoint

```
POST /access/v1/evaluation
```

## Request Format

```json
{
  "subject": {
    "type": "string",
    "id": "string",
    "properties": {
      "key": "value"
    }
  },
  "resource": {
    "type": "string",
    "id": "string",
    "properties": {
      "key": "value"
    }
  },
  "action": {
    "name": "string",
    "properties": {
      "key": "value"
    }
  },
  "context": {
    "key": "value"
  }
}
```

## Response Format

```json
{
  "decision": true|false,
  "context": {
    "key": "value"
  }
}
```

## Examples

### Allow Alice to Read Account 123

**Request:**
```bash
curl -X POST http://localhost:8000/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-001" \
  -d '{
    "subject": {
      "type": "user",
      "id": "alice@acmecorp.com"
    },
    "resource": {
      "type": "account",
      "id": "123"
    },
    "action": {
      "name": "can_read"
    }
  }'
```

**Response:**
```json
{
  "decision": true
}
```

### Manager Updating a Document

**Request:**
```bash
curl -X POST http://localhost:8000/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -d '{
    "subject": {
      "type": "user",
      "id": "bob@example.com",
      "properties": {
        "role": "manager"
      }
    },
    "resource": {
      "type": "document",
      "id": "report-2025"
    },
    "action": {
      "name": "can_update"
    }
  }'
```

**Response:**
```json
{
  "decision": true
}
```

### With Context

**Request:**
```bash
curl -X POST http://localhost:8000/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -d '{
    "subject": {
      "type": "user",
      "id": "carol@example.com"
    },
    "resource": {
      "type": "file",
      "id": "config.txt"
    },
    "action": {
      "name": "can_delete"
    },
    "context": {
      "time": "2025-12-10T15:00:00Z",
      "ip_address": "192.168.1.100"
    }
  }'
```

## Running the Server

SPOCP provides flexible deployment options. The HTTP server always provides monitoring endpoints (`/health`, `/ready`, `/stats`, `/metrics`) and can optionally serve the AuthZen API.

### AuthZen-only (HTTP with monitoring)

Enable AuthZen API on the HTTP server:

```bash
./spocpd -authzen -http-addr :8000 -rules ./examples/rules
```

This starts:
- HTTP server on `:8000` with monitoring endpoints and AuthZen API

### Both TCP and AuthZen

```bash
./spocpd -tcp -tcp-addr :6000 -authzen -http-addr :8000 -rules ./examples/rules
```

This starts:
- TCP server on `:6000` (SPOCP protocol)
- HTTP server on `:8000` with monitoring endpoints and AuthZen API
- Both protocols share the same rule engine

### TCP-only (with HTTP monitoring)

```bash
./spocpd -tcp -tcp-addr :6000 -http-addr :8000 -rules ./examples/rules
```

This starts:
- TCP server on `:6000` (SPOCP protocol)
- HTTP server on `:8000` with monitoring endpoints only (no AuthZen API)

### With All Options

```bash
./spocpd \
  -tcp \
  -tcp-addr :6000 \
  -authzen \
  -http-addr :8000 \
  -rules ./examples/rules \
  -pid /var/run/spocpd.pid \
  -log info \
  -reload 5m
```

Note: At least one of `-tcp` or `-authzen` must be specified to enable a protocol endpoint.

## AuthZen to SPOCP Mapping

AuthZen requests are mapped to SPOCP S-expressions as follows:

### Structure

```
(resource_type (id <id>)(action <action>)(subject (type <type>)(id <id>)(properties...))(context...))
```

### Example Mapping

**AuthZen:**
```json
{
  "subject": {"type": "user", "id": "alice@acmecorp.com"},
  "resource": {"type": "account", "id": "123"},
  "action": {"name": "can_read"}
}
```

**SPOCP S-expression:**
```
(account (id 123)(action can_read)(subject (type user)(id alice@acmecorp.com)))
```

Which in canonical form is:
```
(7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(2:id18:alice@acmecorp.com)))
```

## Writing Rules for AuthZen

Create `.spoc` files with rules that match the AuthZen structure:

```scheme
# examples/rules/authzen.spoc

# Allow alice to read account 123
(7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(2:id18:alice@acmecorp.com)))

# Allow any user in Sales to read account 123
(7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(10:department5:Sales)))

# Allow any manager to update documents
(8:document(6:action10:can_update)(7:subject(4:type4:user)(4:role7:manager)))

# Allow anyone to read public documents
(8:document(10:visibility6:public)(6:action8:can_read)(7:subject))
```

## Common Action Names

The AuthZen specification defines common action names:

- `can_access` - Generic access
- `can_create` - Create new entities
- `can_read` - Read/view content
- `can_update` - Update existing entities
- `can_delete` - Delete entities

You can also define custom actions specific to your application.

## Request Headers

### X-Request-ID

Optional header for request tracking. If provided, the server will include it in the response.

```bash
curl -H "X-Request-ID: unique-request-id" ...
```

The server will echo it back:

```
HTTP/1.1 200 OK
X-Request-ID: unique-request-id
Content-Type: application/json
...
```

## Error Responses

### 400 Bad Request

Invalid JSON or malformed request:

```json
{
  "error": "Bad request: invalid JSON"
}
```

### 405 Method Not Allowed

Using a method other than POST:

```
HTTP/1.1 405 Method Not Allowed
```

### 500 Internal Server Error

Server-side error during evaluation.

## Performance

The HTTP/AuthZen endpoint:
- Shares the same optimized SPOCP engine as the TCP server
- Supports all SPOCP features (star forms, indexing, etc.)
- Uses atomic operations for thread-safe access
- Can handle thousands of requests per second

## Monitoring

The HTTP server always provides monitoring endpoints regardless of whether AuthZen is enabled:

```bash
# Health check
curl http://localhost:8000/health

# Readiness check (checks if rules are loaded)
curl http://localhost:8000/ready

# Statistics in JSON format
curl http://localhost:8000/stats | jq

# Prometheus-style metrics
curl http://localhost:8000/metrics
```

The `/stats` endpoint returns comprehensive metrics for all enabled protocols:

```json
{
  "requests": {
    "tcp": {
      "total": 5678,
      "ok": 5000,
      "deny": 678
    },
    "authzen": {
      "total": 1234,
      "ok": 1100,
      "deny": 134
    }
  },
  "rules": {
    "total": 11,
    "loaded": "2024-01-15T10:30:00Z"
  },
  "indexing": {
    "enabled": true,
    "entries": 42
  }
}
```

## Security Considerations

1. **Always use TLS in production** - Add TLS termination via reverse proxy (nginx, traefik)
2. **Authenticate callers** - Use OAuth 2.0, API keys, or mutual TLS
3. **Rate limiting** - Implement at reverse proxy level
4. **Input validation** - The server validates JSON structure
5. **Network isolation** - Consider running on internal network only

## Integration Examples

### Python

```python
import requests

def check_authorization(subject_id, resource_type, resource_id, action):
    response = requests.post('http://localhost:8000/access/v1/evaluation', json={
        'subject': {
            'type': 'user',
            'id': subject_id
        },
        'resource': {
            'type': resource_type,
            'id': resource_id
        },
        'action': {
            'name': action
        }
    })
    return response.json()['decision']

# Usage
if check_authorization('alice@acmecorp.com', 'account', '123', 'can_read'):
    print("Access granted")
else:
    print("Access denied")
```

### JavaScript/Node.js

```javascript
async function checkAuthorization(subjectId, resourceType, resourceId, action) {
  const response = await fetch('http://localhost:8000/access/v1/evaluation', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Request-ID': crypto.randomUUID()
    },
    body: JSON.stringify({
      subject: { type: 'user', id: subjectId },
      resource: { type: resourceType, id: resourceId },
      action: { name: action }
    })
  });
  
  const result = await response.json();
  return result.decision;
}

// Usage
if (await checkAuthorization('alice@acmecorp.com', 'account', '123', 'can_read')) {
  console.log('Access granted');
}
```

### Go

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

type AuthZenRequest struct {
    Subject  Subject  `json:"subject"`
    Resource Resource `json:"resource"`
    Action   Action   `json:"action"`
}

type AuthZenResponse struct {
    Decision bool `json:"decision"`
}

func checkAuthorization(subjectID, resourceType, resourceID, action string) (bool, error) {
    req := AuthZenRequest{
        Subject:  Subject{Type: "user", ID: subjectID},
        Resource: Resource{Type: resourceType, ID: resourceID},
        Action:   Action{Name: action},
    }
    
    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8000/access/v1/evaluation",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    
    var result AuthZenResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Decision, nil
}
```

## See Also

- [AuthZen Specification](https://openid.net/specs/authorization-api-1_0-01.html)
- [OPERATIONS.md](OPERATIONS.md) - Server operations guide
- [TCP_SERVER.md](TCP_SERVER.md) - TCP protocol documentation
- [SPOCP Specification](docs/draft-hedberg-spocp-sexp-00.txt)
