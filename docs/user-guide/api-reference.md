# API Reference

Complete reference for the LlamaCtl REST API.

## Base URL

All API endpoints are relative to the base URL:

```
http://localhost:8080/api
```

## Authentication

If authentication is enabled, include the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
  http://localhost:8080/api/instances
```

## Instances

### List All Instances

Get a list of all instances.

```http
GET /api/instances
```

**Response:**
```json
{
  "instances": [
    {
      "name": "llama2-7b",
      "status": "running",
      "model_path": "/models/llama-2-7b.gguf",
      "port": 8081,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T12:45:00Z"
    }
  ]
}
```

### Get Instance Details

Get detailed information about a specific instance.

```http
GET /api/instances/{name}
```

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "running",
  "model_path": "/models/llama-2-7b.gguf",
  "port": 8081,
  "pid": 12345,
  "options": {
    "threads": 4,
    "context_size": 2048,
    "gpu_layers": 0
  },
  "stats": {
    "memory_usage": 4294967296,
    "cpu_usage": 25.5,
    "uptime": 3600
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T12:45:00Z"
}
```

### Create Instance

Create a new instance.

```http
POST /api/instances
```

**Request Body:**
```json
{
  "name": "my-instance",
  "model_path": "/path/to/model.gguf",
  "port": 8081,
  "options": {
    "threads": 4,
    "context_size": 2048,
    "gpu_layers": 0
  }
}
```

**Response:**
```json
{
  "message": "Instance created successfully",
  "instance": {
    "name": "my-instance",
    "status": "stopped",
    "model_path": "/path/to/model.gguf",
    "port": 8081,
    "created_at": "2024-01-15T14:30:00Z"
  }
}
```

### Update Instance

Update an existing instance configuration.

```http
PUT /api/instances/{name}
```

**Request Body:**
```json
{
  "options": {
    "threads": 8,
    "context_size": 4096
  }
}
```

### Delete Instance

Delete an instance (must be stopped first).

```http
DELETE /api/instances/{name}
```

**Response:**
```json
{
  "message": "Instance deleted successfully"
}
```

## Instance Operations

### Start Instance

Start a stopped instance.

```http
POST /api/instances/{name}/start
```

**Response:**
```json
{
  "message": "Instance start initiated",
  "status": "starting"
}
```

### Stop Instance

Stop a running instance.

```http
POST /api/instances/{name}/stop
```

**Request Body (Optional):**
```json
{
  "force": false,
  "timeout": 30
}
```

**Response:**
```json
{
  "message": "Instance stop initiated",
  "status": "stopping"
}
```

### Restart Instance

Restart an instance (stop then start).

```http
POST /api/instances/{name}/restart
```

### Get Instance Health

Check instance health status.

```http
GET /api/instances/{name}/health
```

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "process": "running",
    "port": "open",
    "response": "ok"
  },
  "last_check": "2024-01-15T14:30:00Z"
}
```

### Get Instance Logs

Retrieve instance logs.

```http
GET /api/instances/{name}/logs
```

**Query Parameters:**
- `lines`: Number of lines to return (default: 100)
- `follow`: Stream logs (boolean)
- `level`: Filter by log level (debug, info, warn, error)

**Response:**
```json
{
  "logs": [
    {
      "timestamp": "2024-01-15T14:30:00Z",
      "level": "info",
      "message": "Model loaded successfully"
    }
  ]
}
```

## Batch Operations

### Start All Instances

Start all stopped instances.

```http
POST /api/instances/start-all
```

### Stop All Instances

Stop all running instances.

```http
POST /api/instances/stop-all
```

## System Information

### Get System Status

Get overall system status and metrics.

```http
GET /api/system/status
```

**Response:**
```json
{
  "version": "1.0.0",
  "uptime": 86400,
  "instances": {
    "total": 5,
    "running": 3,
    "stopped": 2
  },
  "resources": {
    "cpu_usage": 45.2,
    "memory_usage": 8589934592,
    "memory_total": 17179869184,
    "disk_usage": 75.5
  }
}
```

### Get System Information

Get detailed system information.

```http
GET /api/system/info
```

**Response:**
```json
{
  "hostname": "server-01",
  "os": "linux",
  "arch": "amd64",
  "cpu_count": 8,
  "memory_total": 17179869184,
  "version": "1.0.0",
  "build_time": "2024-01-15T10:00:00Z"
}
```

## Configuration

### Get Configuration

Get current LlamaCtl configuration.

```http
GET /api/config
```

### Update Configuration

Update LlamaCtl configuration (requires restart).

```http
PUT /api/config
```

## Authentication

### Login

Authenticate and receive a JWT token.

```http
POST /api/auth/login
```

**Request Body:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-16T14:30:00Z"
}
```

### Refresh Token

Refresh an existing JWT token.

```http
POST /api/auth/refresh
```

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": "Additional error details"
}
```

### Common HTTP Status Codes

- `200`: Success
- `201`: Created
- `400`: Bad Request
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Not Found
- `409`: Conflict (e.g., instance already exists)
- `500`: Internal Server Error

## WebSocket API

### Real-time Updates

Connect to WebSocket for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws');

ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('Update:', data);
};
```

**Message Types:**
- `instance_status_changed`: Instance status updates
- `instance_stats_updated`: Resource usage updates
- `system_alert`: System-level alerts

## Rate Limiting

API requests are rate limited to:
- **100 requests per minute** for regular endpoints
- **10 requests per minute** for resource-intensive operations

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Reset time (Unix timestamp)

## SDKs and Libraries

### Go Client

```go
import "github.com/lordmathis/llamactl-go-client"

client := llamactl.NewClient("http://localhost:8080")
instances, err := client.ListInstances()
```

### Python Client

```python
from llamactl import Client

client = Client("http://localhost:8080")
instances = client.list_instances()
```

## Examples

### Complete Instance Lifecycle

```bash
# Create instance
curl -X POST http://localhost:8080/api/instances \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example",
    "model_path": "/models/example.gguf",
    "port": 8081
  }'

# Start instance
curl -X POST http://localhost:8080/api/instances/example/start

# Check status
curl http://localhost:8080/api/instances/example

# Stop instance
curl -X POST http://localhost:8080/api/instances/example/stop

# Delete instance
curl -X DELETE http://localhost:8080/api/instances/example
```

## Next Steps

- Learn about [Managing Instances](managing-instances.md) in detail
- Explore [Advanced Configuration](../advanced/backends.md)
- Set up [Monitoring](../advanced/monitoring.md) for production use
