# llamactl

![Build and Release](https://github.com/lordmathis/llamactl/actions/workflows/release.yaml/badge.svg) ![Go Tests](https://github.com/lordmathis/llamactl/actions/workflows/go_test.yaml/badge.svg) ![WebUI Tests](https://github.com/lordmathis/llamactl/actions/workflows/webui_test.yaml/badge.svg)

A control server for managing multiple Llama Server instances with a web-based dashboard.

## Features

- **Multi-instance Management**: Create, start, stop, restart, and delete multiple llama-server instances
- **Web Dashboard**: Modern React-based UI for managing instances
- **Auto-restart**: Configurable automatic restart on instance failure
- **Instance Monitoring**: Real-time health checks and status monitoring
- **Log Management**: View, search, and download instance logs
- **REST API**: Full API for programmatic control
- **OpenAI Compatible**: Route requests to instances by instance name
- **Configuration Management**: Comprehensive llama-server parameter support
- **System Information**: View llama-server version, devices, and help

## Prerequisites

This project requires `llama-server` from llama.cpp to be installed and available in your PATH.

**Install llama.cpp:**
Follow the installation instructions at https://github.com/ggml-org/llama.cpp

## Installation

### Build Requirements

- Go 1.24 or later
- Node.js 22 or later (for building the web UI)

### Building with Web UI

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

Configuration files are searched in the following locations:

**Linux/macOS:**
- `./llamactl.yaml` or `./config.yaml` (current directory)
- `~/.config/llamactl/config.yaml`
- `/etc/llamactl/config.yaml`

**Windows:**
- `./llamactl.yaml` or `./config.yaml` (current directory)
- `%APPDATA%\llamactl\config.yaml`
- `%PROGRAMDATA%\llamactl\config.yaml`

You can specify the path to config file with `LLAMACTL_CONFIG_PATH` environment variable

### Configuration Options

#### Server Configuration

```yaml
server:
  host: ""              # Server host to bind to (default: "")
  port: 8080             # Server port to bind to (default: 8080)
```

**Environment Variables:**
- `LLAMACTL_HOST` - Server host
- `LLAMACTL_PORT` - Server port

#### Instance Configuration

```yaml
instances:
  port_range: [8000, 9000]           # Port range for instances
  log_directory: "/tmp/llamactl"     # Directory for instance logs
  max_instances: -1                  # Maximum instances (-1 = unlimited)
  llama_executable: "llama-server"   # Path to llama-server executable
  default_auto_restart: true         # Default auto-restart setting
  default_max_restarts: 3            # Default maximum restart attempts
  default_restart_delay: 5           # Default restart delay in seconds
```

**Environment Variables:**
- `LLAMACTL_INSTANCE_PORT_RANGE` - Port range (format: "8000-9000" or "8000,9000")
- `LLAMACTL_LOG_DIR` - Log directory path
- `LLAMACTL_MAX_INSTANCES` - Maximum number of instances
- `LLAMACTL_LLAMA_EXECUTABLE` - Path to llama-server executable
- `LLAMACTL_DEFAULT_AUTO_RESTART` - Default auto-restart setting (true/false)
- `LLAMACTL_DEFAULT_MAX_RESTARTS` - Default maximum restarts
- `LLAMACTL_DEFAULT_RESTART_DELAY` - Default restart delay in seconds

### Example Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080

instances:
  port_range: [8001, 8100]
  log_directory: "/var/log/llamactl"
  max_instances: 10
  llama_executable: "/usr/local/bin/llama-server"
  default_auto_restart: true
  default_max_restarts: 5
  default_restart_delay: 10
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