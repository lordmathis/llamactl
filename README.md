# llamactl

![Build and Release](https://github.com/lordmathis/llamactl/actions/workflows/release.yaml/badge.svg) ![Go Tests](https://github.com/lordmathis/llamactl/actions/workflows/go_test.yaml/badge.svg) ![WebUI Tests](https://github.com/lordmathis/llamactl/actions/workflows/webui_test.yaml/badge.svg)

A control server for managing multiple Llama Server instances with a web-based dashboard.

## Features

- **Multi-instance Management**: Create, start, stop, restart, and delete multiple llama-server instances
- **Web Dashboard**: Modern React-based UI for managing instances
- **Auto-restart**: Configurable automatic restart on instance failure
- **Instance Monitoring**: Real-time health checks and status monitoring
- **Log Management**: View, search, and download instance logs
- **Data Persistence**: Persistent storage of instance state.
- **REST API**: Full API for programmatic control
- **OpenAI Compatible**: Route requests to instances by instance name
- **Configuration Management**: Comprehensive llama-server parameter support
- **System Information**: View llama-server version, devices, and help

## Prerequisites

This project requires `llama-server` from llama.cpp to be installed and available in your PATH.

**Install llama.cpp:**
Follow the installation instructions at https://github.com/ggml-org/llama.cpp

## Installation

### Download Prebuilt Binaries

The easiest way to install llamactl is to download a prebuilt binary from the [releases page](https://github.com/lordmathis/llamactl/releases).

**Linux/macOS:**
```bash
# Download the latest release for your platform
curl -L https://github.com/lordmathis/llamactl/releases/latest/download/llamactl-$(curl -s https://api.github.com/repos/lordmathis/llamactl/releases/latest | grep tag_name | cut -d '"' -f 4)-linux-amd64.tar.gz | tar -xz

# Move to PATH
sudo mv llamactl /usr/local/bin/

# Run the server
llamactl
```

**Manual Download:**
1. Go to the [releases page](https://github.com/lordmathis/llamactl/releases)
2. Download the appropriate archive for your platform
3. Extract the archive and move the binary to a directory in your PATH

### Build from Source

If you prefer to build from source or need the latest development version:

#### Build Requirements

- Go 1.24 or later
- Node.js 22 or later (for building the web UI)

#### Building with Web UI

```bash
# Clone the repository
git clone https://github.com/lordmathis/llamactl.git
cd llamactl

# Install Node.js dependencies
cd webui
npm ci

# Build the web UI
npm run build

# Return to project root and build
cd ..
go build -o llamactl ./cmd/server

# Run the server
./llamactl
```

## Configuration

llamactl can be configured via configuration files or environment variables. Configuration is loaded in the following order of precedence:

1. Hardcoded defaults
2. Configuration file
3. Environment variables


### Configuration Files


#### Configuration File Locations

Configuration files are searched in the following locations (in order of precedence):

**Linux/macOS:**
- `./llamactl.yaml` or `./config.yaml` (current directory)
- `$HOME/.config/llamactl/config.yaml`
- `/etc/llamactl/config.yaml`

**Windows:**
- `./llamactl.yaml` or `./config.yaml` (current directory)
- `%APPDATA%\llamactl\config.yaml`
- `%USERPROFILE%\llamactl\config.yaml`
- `%PROGRAMDATA%\llamactl\config.yaml`

You can specify the path to config file with `LLAMACTL_CONFIG_PATH` environment variable.

## API Key Authentication

llamactl now supports API Key authentication for both management and inference (OpenAI-compatible) endpoints. The are separate keys for management and inference APIs. Management keys grant full access; inference keys grant access to OpenAI-compatible endpoints

**How to Use:**
- Pass your API key in requests using one of:
  - `Authorization: Bearer <key>` header
  - `X-API-Key: <key>` header
  - `api_key=<key>` query parameter

**Auto-generated keys**: If no keys are set and authentication is required, a key will be generated and printed to the terminal at startup. For production, set your own keys in config or environment variables.

### Configuration Options


#### Server Configuration

```yaml
server:
  host: "0.0.0.0"         # Server host to bind to (default: "0.0.0.0")
  port: 8080              # Server port to bind to (default: 8080)
  allowed_origins: ["*"]  # CORS allowed origins (default: ["*"])
  enable_swagger: false   # Enable Swagger UI (default: false)
```

**Environment Variables:**
- `LLAMACTL_HOST` - Server host
- `LLAMACTL_PORT` - Server port
- `LLAMACTL_ALLOWED_ORIGINS` - Comma-separated CORS origins
- `LLAMACTL_ENABLE_SWAGGER` - Enable Swagger UI (true/false)


#### Instance Configuration

```yaml
instances:
  port_range: [8000, 9000]           # Port range for instances (default: [8000, 9000])
  data_dir: "~/.local/share/llamactl" # Directory for all llamactl data (default varies by OS)
  configs_dir: "~/.local/share/llamactl/instances" # Directory for instance configs (default: data_dir/instances)
  logs_dir: "~/.local/share/llamactl/logs" # Directory for instance logs (default: data_dir/logs)
  auto_create_dirs: true             # Automatically create data/config/logs directories (default: true)
  max_instances: -1                  # Maximum instances (-1 = unlimited)
  llama_executable: "llama-server"   # Path to llama-server executable
  default_auto_restart: true         # Default auto-restart setting
  default_max_restarts: 3            # Default maximum restart attempts
  default_restart_delay: 5           # Default restart delay in seconds
```

**Environment Variables:**
- `LLAMACTL_INSTANCE_PORT_RANGE` - Port range (format: "8000-9000" or "8000,9000")
- `LLAMACTL_DATA_DIRECTORY` - Data directory path
- `LLAMACTL_INSTANCES_DIR` - Instance configs directory path
- `LLAMACTL_LOGS_DIR` - Log directory path
- `LLAMACTL_AUTO_CREATE_DATA_DIR` - Auto-create data/config/logs directories (true/false)
- `LLAMACTL_MAX_INSTANCES` - Maximum number of instances
- `LLAMACTL_LLAMA_EXECUTABLE` - Path to llama-server executable
- `LLAMACTL_DEFAULT_AUTO_RESTART` - Default auto-restart setting (true/false)
- `LLAMACTL_DEFAULT_MAX_RESTARTS` - Default maximum restarts
- `LLAMACTL_DEFAULT_RESTART_DELAY` - Default restart delay in seconds


#### Auth Configuration

```yaml
auth:
  require_inference_auth: true           # Require API key for OpenAI endpoints (default: true)
  inference_keys: []                     # List of valid inference API keys
  require_management_auth: true          # Require API key for management endpoints (default: true)
  management_keys: []                    # List of valid management API keys
```

**Environment Variables:**
- `LLAMACTL_REQUIRE_INFERENCE_AUTH` - Require auth for OpenAI endpoints (true/false)
- `LLAMACTL_INFERENCE_KEYS` - Comma-separated inference API keys
- `LLAMACTL_REQUIRE_MANAGEMENT_AUTH` - Require auth for management endpoints (true/false)
- `LLAMACTL_MANAGEMENT_KEYS` - Comma-separated management API keys


### Example Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080

instances:
  port_range: [8001, 8100]
  data_dir: "/var/lib/llamactl"
  configs_dir: "/var/lib/llamactl/instances"
  logs_dir: "/var/log/llamactl"
  auto_create_dirs: true
  max_instances: 10
  llama_executable: "/usr/local/bin/llama-server"
  default_auto_restart: true
  default_max_restarts: 5
  default_restart_delay: 10

auth:
  require_inference_auth: true
  inference_keys: ["sk-inference-abc123"]
  require_management_auth: true
  management_keys: ["sk-management-xyz456"]
```

## Usage

### Starting the Server

```bash
# Start with default configuration
./llamactl

# Start with custom config file
LLAMACTL_CONFIG_PATH=/path/to/config.yaml ./llamactl

# Start with environment variables
LLAMACTL_PORT=9090 LLAMACTL_LOG_DIR=/custom/logs ./llamactl
```

### Web Dashboard

Open your browser and navigate to `http://localhost:8080` to access the web dashboard.

### API Usage

The REST API is available at `http://localhost:8080/api/v1`. See the Swagger documentation at `http://localhost:8080/swagger/` for complete API reference.

#### Create an Instance

```bash
curl -X POST http://localhost:8080/api/v1/instances/my-instance \
  -H "Content-Type: application/json" \
  -d '{
    "model": "/path/to/model.gguf",
    "gpu_layers": 32,
    "auto_restart": true
  }'
```

#### List Instances

```bash
curl http://localhost:8080/api/v1/instances
```

#### Start/Stop Instance

```bash
# Start
curl -X POST http://localhost:8080/api/v1/instances/my-instance/start

# Stop
curl -X POST http://localhost:8080/api/v1/instances/my-instance/stop
```

### OpenAI Compatible Endpoints

Route requests to instances by including the instance name as the model parameter:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-instance",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Development

### Running Tests

```bash
# Go tests
go test ./...

# Web UI tests
cd webui
npm test
```

### Development Server

```bash
# Start Go server in development mode
go run ./cmd/server

# Start web UI development server (in another terminal)
cd webui
npm run dev
```

## API Documentation

Interactive API documentation is available at `http://localhost:8080/swagger/` when the server is running.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.