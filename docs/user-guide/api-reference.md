# API Reference

Complete reference for the Llamactl REST API.

## Base URL

All API endpoints are relative to the base URL:

```
http://localhost:8080/api/v1
```

## Authentication

Llamactl supports API key authentication. If authentication is enabled, include the API key in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-api-key>" \
  http://localhost:8080/api/v1/instances
```

The server supports two types of API keys:
- **Management API Keys**: Required for instance management operations (CRUD operations on instances)
- **Inference API Keys**: Required for OpenAI-compatible inference endpoints

## System Endpoints

### Get Llamactl Version

Get the version information of the llamactl server.

```http
GET /api/v1/version
```

**Response:**
```
Version: 1.0.0
Commit: abc123
Build Time: 2024-01-15T10:00:00Z
```

### Get Llama Server Help

Get help text for the llama-server command.

```http
GET /api/v1/server/help
```

**Response:** Plain text help output from `llama-server --help`

### Get Llama Server Version

Get version information of the llama-server binary.

```http
GET /api/v1/server/version
```

**Response:** Plain text version output from `llama-server --version`

### List Available Devices

List available devices for llama-server.

```http
GET /api/v1/server/devices
```

**Response:** Plain text device list from `llama-server --list-devices`

## Instances

### List All Instances

Get a list of all instances.

```http
GET /api/v1/instances
```

**Response:**
```json
[
  {
    "name": "llama2-7b",
    "status": "running",
    "created": 1705312200
  }
]
```

### Get Instance Details

Get detailed information about a specific instance.

```http
GET /api/v1/instances/{name}
```

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "running",
  "created": 1705312200
}
```

### Create Instance

Create and start a new instance.

```http
POST /api/v1/instances/{name}
```

**Request Body:** JSON object with instance configuration. See [Managing Instances](managing-instances.md) for available configuration options.

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "running",
  "created": 1705312200
}
```

### Update Instance

Update an existing instance configuration. See [Managing Instances](managing-instances.md) for available configuration options.

```http
PUT /api/v1/instances/{name}
```

**Request Body:** JSON object with configuration fields to update.

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "running",
  "created": 1705312200
}
```

### Delete Instance

Stop and remove an instance.

```http
DELETE /api/v1/instances/{name}
```

**Response:** `204 No Content`

## Instance Operations

### Start Instance

Start a stopped instance.

```http
POST /api/v1/instances/{name}/start
```

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "starting",
  "created": 1705312200
}
```

**Error Responses:**
- `409 Conflict`: Maximum number of running instances reached
- `500 Internal Server Error`: Failed to start instance

### Stop Instance

Stop a running instance.

```http
POST /api/v1/instances/{name}/stop
```

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "stopping",
  "created": 1705312200
}
```

### Restart Instance

Restart an instance (stop then start).

```http
POST /api/v1/instances/{name}/restart
```

**Response:**
```json
{
  "name": "llama2-7b",
  "status": "restarting",
  "created": 1705312200
}
```

### Get Instance Logs

Retrieve instance logs.

```http
GET /api/v1/instances/{name}/logs
```

**Query Parameters:**
- `lines`: Number of lines to return (default: all lines, use -1 for all)

**Response:** Plain text log output

**Example:**
```bash
curl "http://localhost:8080/api/v1/instances/my-instance/logs?lines=100"
```

### Proxy to Instance

Proxy HTTP requests directly to the llama-server instance.

```http
GET /api/v1/instances/{name}/proxy/*
POST /api/v1/instances/{name}/proxy/*
```

This endpoint forwards all requests to the underlying llama-server instance running on its configured port. The proxy strips the `/api/v1/instances/{name}/proxy` prefix and forwards the remaining path to the instance.

**Example - Check Instance Health:**
```bash
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/api/v1/instances/my-model/proxy/health
```

This forwards the request to `http://instance-host:instance-port/health` on the actual llama-server instance.

**Error Responses:**
- `503 Service Unavailable`: Instance is not running

## OpenAI-Compatible API

Llamactl provides OpenAI-compatible endpoints for inference operations.

### List Models

List all instances in OpenAI-compatible format.

```http
GET /v1/models
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "llama2-7b",
      "object": "model",
      "created": 1705312200,
      "owned_by": "llamactl"
    }
  ]
}
```

### Chat Completions, Completions, Embeddings

All OpenAI-compatible inference endpoints are available:

```http
POST /v1/chat/completions
POST /v1/completions
POST /v1/embeddings
POST /v1/rerank
POST /v1/reranking
```

**Request Body:** Standard OpenAI format with `model` field specifying the instance name

**Example:**
```json
{
  "model": "llama2-7b",
  "messages": [
    {
      "role": "user",
      "content": "Hello, how are you?"
    }
  ]
}
```

The server routes requests to the appropriate instance based on the `model` field in the request body. Instances with on-demand starting enabled will be automatically started if not running. For configuration details, see [Managing Instances](managing-instances.md).

**Error Responses:**
- `400 Bad Request`: Invalid request body or missing model name
- `503 Service Unavailable`: Instance is not running and on-demand start is disabled
- `409 Conflict`: Cannot start instance due to maximum instances limit

## Instance Status Values

Instances can have the following status values:
- `stopped`: Instance is not running
- `running`: Instance is running and ready to accept requests
- `failed`: Instance failed to start or crashed

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": "Error message description"
}
```

### Common HTTP Status Codes

- `200`: Success
- `201`: Created
- `204`: No Content (successful deletion)
- `400`: Bad Request (invalid parameters or request body)
- `401`: Unauthorized (missing or invalid API key)
- `403`: Forbidden (insufficient permissions)
- `404`: Not Found (instance not found)
- `409`: Conflict (instance already exists, max instances reached)
- `500`: Internal Server Error
- `503`: Service Unavailable (instance not running)

## Examples

### Complete Instance Lifecycle

```bash
# Create and start instance
curl -X POST http://localhost:8080/api/v1/instances/my-model \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "/models/llama-2-7b.gguf"
  }'

# Check instance status
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/api/v1/instances/my-model

# Get instance logs
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8080/api/v1/instances/my-model/logs?lines=50"

# Use OpenAI-compatible chat completions
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-inference-api-key" \
  -d '{
    "model": "my-model",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "max_tokens": 100
  }'

# Stop instance
curl -X POST -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/api/v1/instances/my-model/stop

# Delete instance
curl -X DELETE -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/api/v1/instances/my-model
```

### Using the Proxy Endpoint

You can also directly proxy requests to the llama-server instance:

```bash
# Direct proxy to instance (bypasses OpenAI compatibility layer)
curl -X POST http://localhost:8080/api/v1/instances/my-model/proxy/completion \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "prompt": "Hello, world!",
    "n_predict": 50
  }'
```

## Swagger Documentation

If swagger documentation is enabled in the server configuration, you can access the interactive API documentation at:

```
http://localhost:8080/swagger/
```

This provides a complete interactive interface for testing all API endpoints.
