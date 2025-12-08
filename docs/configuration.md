# Configuration

llamactl can be configured via configuration files or environment variables. Configuration is loaded in the following order of precedence:

```
Defaults < Configuration file < Environment variables
```

llamactl works out of the box with sensible defaults, but you can customize the behavior to suit your needs.

## Default Configuration

Here's the default configuration with all available options:

```yaml
server:
  host: "0.0.0.0"                # Server host to bind to
  port: 8080                     # Server port to bind to
  allowed_origins: ["*"]         # Allowed CORS origins (default: all)
  allowed_headers: ["*"]         # Allowed CORS headers (default: all)
  enable_swagger: false          # Enable Swagger UI for API docs

backends:
  llama-cpp:
    command: "llama-server"
    args: []
    environment: {}              # Environment variables for the backend process
    docker:
      enabled: false
      image: "ghcr.io/ggml-org/llama.cpp:server"
      args: ["run", "--rm", "--network", "host", "--gpus", "all"]
      environment: {}
    response_headers: {}         # Additional response headers to send with responses

  vllm:
    command: "vllm"
    args: ["serve"]
    environment: {}              # Environment variables for the backend process
    docker:
      enabled: false
      image: "vllm/vllm-openai:latest"
      args: ["run", "--rm", "--network", "host", "--gpus", "all", "--shm-size", "1g"]
      environment: {}
    response_headers: {}         # Additional response headers to send with responses

  mlx:
    command: "mlx_lm.server"
    args: []
    environment: {}              # Environment variables for the backend process
    response_headers: {}         # Additional response headers to send with responses

data_dir: ~/.local/share/llamactl  # Main data directory (database, instances, logs), default varies by OS

instances:
  port_range: [8000, 9000]         # Port range for instances
  configs_dir: data_dir/instances  # Instance configs directory
  logs_dir: data_dir/logs          # Logs directory
  auto_create_dirs: true           # Auto-create data/config/logs dirs if missing
  max_instances: -1                # Max instances (-1 = unlimited)
  max_running_instances: -1        # Max running instances (-1 = unlimited)
  enable_lru_eviction: true        # Enable LRU eviction for idle instances
  default_auto_restart: true       # Auto-restart new instances by default
  default_max_restarts: 3          # Max restarts for new instances
  default_restart_delay: 5         # Restart delay (seconds) for new instances
  default_on_demand_start: true    # Default on-demand start setting
  on_demand_start_timeout: 120     # Default on-demand start timeout in seconds
  timeout_check_interval: 5        # Idle instance timeout check in minutes

database:
  path: data_dir/llamactl.db              # Database file path
  max_open_connections: 25       # Maximum open database connections
  max_idle_connections: 5        # Maximum idle database connections
  connection_max_lifetime: 5m    # Connection max lifetime

auth:
  require_inference_auth: true   # Require auth for inference endpoints
  require_management_auth: true  # Require auth for management endpoints
  management_keys: []            # Keys for management endpoints

local_node: "main"               # Name of the local node (default: "main")
nodes:                           # Node configuration for multi-node deployment
  main:                          # Default local node (empty config)
```

## Configuration Files

### Configuration File Locations

Configuration files are searched in the following locations (in order of precedence, first found is used):

**Linux:**  
- `./llamactl.yaml` or `./config.yaml` (current directory)  
- `$HOME/.config/llamactl/config.yaml`  
- `/etc/llamactl/config.yaml`  

**macOS:**  
- `./llamactl.yaml` or `./config.yaml` (current directory)  
- `$HOME/Library/Application Support/llamactl/config.yaml`  
- `/Library/Application Support/llamactl/config.yaml`  

**Windows:**  
- `./llamactl.yaml` or `./config.yaml` (current directory)  
- `%APPDATA%\llamactl\config.yaml`  
- `%USERPROFILE%\llamactl\config.yaml`  
- `%PROGRAMDATA%\llamactl\config.yaml`  

You can specify the path to config file with `LLAMACTL_CONFIG_PATH` environment variable.

## Configuration Options

### Server Configuration

```yaml
server:
  host: "0.0.0.0"         # Server host to bind to (default: "0.0.0.0")
  port: 8080              # Server port to bind to (default: 8080)
  allowed_origins: ["*"]  # CORS allowed origins (default: ["*"])
  allowed_headers: ["*"]  # CORS allowed headers (default: ["*"])
  enable_swagger: false   # Enable Swagger UI (default: false)
```

**Environment Variables:**
- `LLAMACTL_HOST` - Server host
- `LLAMACTL_PORT` - Server port
- `LLAMACTL_ALLOWED_ORIGINS` - Comma-separated CORS origins
- `LLAMACTL_ENABLE_SWAGGER` - Enable Swagger UI (true/false)

### Backend Configuration
```yaml
backends:
  llama-cpp:
    command: "llama-server"
    args: []
    environment: {}              # Environment variables for the backend process
    docker:
      enabled: false             # Enable Docker runtime (default: false)
      image: "ghcr.io/ggml-org/llama.cpp:server"
      args: ["run", "--rm", "--network", "host", "--gpus", "all"]
      environment: {}
    response_headers: {}         # Additional response headers to send with responses

  vllm:
    command: "vllm"
    args: ["serve"]
    environment: {}              # Environment variables for the backend process
    docker:
      enabled: false             # Enable Docker runtime (default: false)
      image: "vllm/vllm-openai:latest"
      args: ["run", "--rm", "--network", "host", "--gpus", "all", "--shm-size", "1g"]
      environment: {}
    response_headers: {}         # Additional response headers to send with responses

  mlx:
    command: "mlx_lm.server"
    args: []
    environment: {}              # Environment variables for the backend process
    # MLX does not support Docker
    response_headers: {}         # Additional response headers to send with responses
```

**Backend Configuration Fields:**
- `command`: Executable name/path for the backend
- `args`: Default arguments prepended to all instances
- `environment`: Environment variables for the backend process (optional)
- `response_headers`: Additional response headers to send with responses (optional)
- `docker`: Docker-specific configuration (optional)
  - `enabled`: Boolean flag to enable Docker runtime
  - `image`: Docker image to use
  - `args`: Additional arguments passed to `docker run`
  - `environment`: Environment variables for the container (optional)

> If llamactl is behind an NGINX proxy, `X-Accel-Buffering: no` response header may be required for NGINX to properly stream the responses without buffering.

**Environment Variables:**

**LlamaCpp Backend:**
- `LLAMACTL_LLAMACPP_COMMAND` - LlamaCpp executable command
- `LLAMACTL_LLAMACPP_ARGS` - Space-separated default arguments
- `LLAMACTL_LLAMACPP_ENV` - Environment variables in format "KEY1=value1,KEY2=value2"
- `LLAMACTL_LLAMACPP_DOCKER_ENABLED` - Enable Docker runtime (true/false)
- `LLAMACTL_LLAMACPP_DOCKER_IMAGE` - Docker image to use
- `LLAMACTL_LLAMACPP_DOCKER_ARGS` - Space-separated Docker arguments
- `LLAMACTL_LLAMACPP_DOCKER_ENV` - Docker environment variables in format "KEY1=value1,KEY2=value2"
- `LLAMACTL_LLAMACPP_RESPONSE_HEADERS` - Response headers in format "KEY1=value1;KEY2=value2"

**VLLM Backend:**
- `LLAMACTL_VLLM_COMMAND` - VLLM executable command
- `LLAMACTL_VLLM_ARGS` - Space-separated default arguments
- `LLAMACTL_VLLM_ENV` - Environment variables in format "KEY1=value1,KEY2=value2"
- `LLAMACTL_VLLM_DOCKER_ENABLED` - Enable Docker runtime (true/false)
- `LLAMACTL_VLLM_DOCKER_IMAGE` - Docker image to use
- `LLAMACTL_VLLM_DOCKER_ARGS` - Space-separated Docker arguments
- `LLAMACTL_VLLM_DOCKER_ENV` - Docker environment variables in format "KEY1=value1,KEY2=value2"
- `LLAMACTL_VLLM_RESPONSE_HEADERS` - Response headers in format "KEY1=value1;KEY2=value2"

**MLX Backend:**
- `LLAMACTL_MLX_COMMAND` - MLX executable command
- `LLAMACTL_MLX_ARGS` - Space-separated default arguments
- `LLAMACTL_MLX_ENV` - Environment variables in format "KEY1=value1,KEY2=value2"
- `LLAMACTL_MLX_RESPONSE_HEADERS` - Response headers in format "KEY1=value1;KEY2=value2"

### Data Directory Configuration

```yaml
data_dir: "~/.local/share/llamactl"  # Main data directory for database, instances, and logs (default varies by OS)
```

**Environment Variables:**
- `LLAMACTL_DATA_DIRECTORY` - Main data directory path

**Default Data Directory by Platform:**
- **Linux**: `~/.local/share/llamactl`
- **macOS**: `~/Library/Application Support/llamactl`
- **Windows**: `%LOCALAPPDATA%\llamactl` or `%PROGRAMDATA%\llamactl`

### Instance Configuration

```yaml
instances:
  port_range: [8000, 9000]      # Port range for instances (default: [8000, 9000])
  configs_dir: "instances"      # Directory for instance configs, default: data_dir/instances
  logs_dir: "logs"              # Directory for instance logs, default: data_dir/logs
  auto_create_dirs: true        # Automatically create data/config/logs directories (default: true)
  max_instances: -1             # Maximum instances (-1 = unlimited)
  max_running_instances: -1     # Maximum running instances (-1 = unlimited)
  enable_lru_eviction: true     # Enable LRU eviction for idle instances
  default_auto_restart: true    # Default auto-restart setting
  default_max_restarts: 3       # Default maximum restart attempts
  default_restart_delay: 5      # Default restart delay in seconds
  default_on_demand_start: true # Default on-demand start setting
  on_demand_start_timeout: 120  # Default on-demand start timeout in seconds
  timeout_check_interval: 5     # Default instance timeout check interval in minutes
```

**Environment Variables:**
- `LLAMACTL_INSTANCE_PORT_RANGE` - Port range (format: "8000-9000" or "8000,9000")
- `LLAMACTL_INSTANCES_DIR` - Instance configs directory path
- `LLAMACTL_LOGS_DIR` - Log directory path
- `LLAMACTL_AUTO_CREATE_DATA_DIR` - Auto-create data/config/logs directories (true/false)
- `LLAMACTL_MAX_INSTANCES` - Maximum number of instances  
- `LLAMACTL_MAX_RUNNING_INSTANCES` - Maximum number of running instances
- `LLAMACTL_ENABLE_LRU_EVICTION` - Enable LRU eviction for idle instances
- `LLAMACTL_DEFAULT_AUTO_RESTART` - Default auto-restart setting (true/false)  
- `LLAMACTL_DEFAULT_MAX_RESTARTS` - Default maximum restarts  
- `LLAMACTL_DEFAULT_RESTART_DELAY` - Default restart delay in seconds  
- `LLAMACTL_DEFAULT_ON_DEMAND_START` - Default on-demand start setting (true/false)  
- `LLAMACTL_ON_DEMAND_START_TIMEOUT` - Default on-demand start timeout in seconds
- `LLAMACTL_TIMEOUT_CHECK_INTERVAL` - Default instance timeout check interval in minutes

### Database Configuration

```yaml
database:
  path: "llamactl.db"              # Database file path, default: data_dir/llamactl.db
  max_open_connections: 25         # Maximum open database connections (default: 25)
  max_idle_connections: 5          # Maximum idle database connections (default: 5)
  connection_max_lifetime: 5m      # Connection max lifetime (default: 5m)
```

**Environment Variables:**
- `LLAMACTL_DATABASE_PATH` - Database file path (relative to data_dir or absolute)
- `LLAMACTL_DATABASE_MAX_OPEN_CONNECTIONS` - Maximum open database connections
- `LLAMACTL_DATABASE_MAX_IDLE_CONNECTIONS` - Maximum idle database connections
- `LLAMACTL_DATABASE_CONN_MAX_LIFETIME` - Connection max lifetime (e.g., "5m", "1h")

### Authentication Configuration

llamactl supports two types of authentication:

- **Management API Keys**: For accessing the web UI and management API (creating/managing instances). These can be configured in the config file or via environment variables.
- **Inference API Keys**: For accessing the OpenAI-compatible inference endpoints. These are managed via the web UI (Settings → API Keys) and stored in the database.

```yaml
auth:
  require_inference_auth: true           # Require API key for OpenAI endpoints (default: true)
  require_management_auth: true          # Require API key for management endpoints (default: true)
  management_keys: []                    # List of valid management API keys
```

**Managing Inference API Keys:**

Inference API keys are managed through the web UI or management API and stored in the database. To create and manage inference keys:

1. Open the web UI and log in with a management API key
2. Navigate to **Settings → API Keys**
3. Click **Create API Key**
4. Configure the key:
   - **Name**: A descriptive name for the key
   - **Expiration**: Optional expiration date
   - **Permissions**: Grant access to all instances or specific instances only
5. Copy the generated key - it won't be shown again

**Environment Variables:**
- `LLAMACTL_REQUIRE_INFERENCE_AUTH` - Require auth for OpenAI endpoints (true/false)
- `LLAMACTL_REQUIRE_MANAGEMENT_AUTH` - Require auth for management endpoints (true/false)
- `LLAMACTL_MANAGEMENT_KEYS` - Comma-separated management API keys

### Remote Node Configuration

llamactl supports remote node deployments. Configure remote nodes to deploy instances on remote hosts and manage them centrally.

```yaml
local_node: "main"               # Name of the local node (default: "main")
nodes:                           # Node configuration map
  main:                          # Local node (empty address means local)
    address: ""                  # Not used for local node
    api_key: ""                  # Not used for local node
  worker1:                       # Remote worker node
    address: "http://192.168.1.10:8080"
    api_key: "worker1-api-key"   # Management API key for authentication
```

**Node Configuration Fields:**
- `local_node`: Specifies which node in the `nodes` map represents the local node. Must match exactly what other nodes call this node.
- `nodes`: Map of node configurations
  - `address`: HTTP/HTTPS URL of the remote node (empty for local node)
  - `api_key`: Management API key for authenticating with the remote node

**Environment Variables:**
- `LLAMACTL_LOCAL_NODE` - Name of the local node
