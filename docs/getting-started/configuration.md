# Configuration

Llamactl can be configured through various methods to suit your needs.

## Configuration File

Create a configuration file at `~/.llamactl/config.yaml`:

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  cors_enabled: true

# Authentication (optional)
auth:
  enabled: false
  # When enabled, configure your authentication method
  # jwt_secret: "your-secret-key"

# Default instance settings
defaults:
  backend: "llamacpp"
  timeout: 300
  log_level: "info"

# Paths
paths:
  models_dir: "/path/to/your/models"
  logs_dir: "/var/log/llamactl"
  data_dir: "/var/lib/llamactl"

# Instance limits
limits:
  max_instances: 10
  max_memory_per_instance: "8GB"
```

## Environment Variables

You can also configure Llamactl using environment variables:

```bash
# Server settings
export LLAMACTL_HOST=0.0.0.0
export LLAMACTL_PORT=8080

# Paths
export LLAMACTL_MODELS_DIR=/path/to/models
export LLAMACTL_LOGS_DIR=/var/log/llamactl

# Limits
export LLAMACTL_MAX_INSTANCES=5
```

## Command Line Options

View all available command line options:

```bash
llamactl --help
```

Common options:

```bash
# Specify config file
llamactl --config /path/to/config.yaml

# Set log level
llamactl --log-level debug

# Run on different port
llamactl --port 9090
```

## Instance Configuration

When creating instances, you can specify various options:

### Basic Options

- `name`: Unique identifier for the instance
- `model_path`: Path to the GGUF model file
- `port`: Port for the instance to listen on

### Advanced Options

- `threads`: Number of CPU threads to use
- `context_size`: Context window size
- `batch_size`: Batch size for processing
- `gpu_layers`: Number of layers to offload to GPU
- `memory_lock`: Lock model in memory
- `no_mmap`: Disable memory mapping

### Example Instance Configuration

```json
{
  "name": "production-model",
  "model_path": "/models/llama-2-13b-chat.gguf",
  "port": 8081,
  "options": {
    "threads": 8,
    "context_size": 4096,
    "batch_size": 512,
    "gpu_layers": 35,
    "memory_lock": true
  }
}
```

## Security Configuration

### Enable Authentication

To enable authentication, update your config file:

```yaml
auth:
  enabled: true
  jwt_secret: "your-very-secure-secret-key"
  token_expiry: "24h"
```

### HTTPS Configuration

For production deployments, configure HTTPS:

```yaml
server:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
```

## Logging Configuration

Configure logging levels and outputs:

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or text
  output: "/var/log/llamactl/app.log"
```

## Next Steps

- Learn about [Managing Instances](../user-guide/managing-instances.md)
- Explore [Advanced Configuration](../advanced/monitoring.md)
- Set up [Monitoring](../advanced/monitoring.md)
