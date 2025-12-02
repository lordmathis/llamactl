# llamactl

![Build and Release](https://github.com/lordmathis/llamactl/actions/workflows/release.yaml/badge.svg) ![Go Tests](https://github.com/lordmathis/llamactl/actions/workflows/go_test.yaml/badge.svg) ![WebUI Tests](https://github.com/lordmathis/llamactl/actions/workflows/webui_test.yaml/badge.svg) ![User Docs](https://github.com/lordmathis/llamactl/actions/workflows/docs.yaml/badge.svg)

**Unified management and routing for llama.cpp, MLX and vLLM models with web dashboard.**

📚 **[Full Documentation →](https://llamactl.org)**

![Dashboard Screenshot](docs/images/dashboard.png)

## Features

**🚀 Easy Model Management**
- **Multiple Models Simultaneously**: Run different models at the same time (7B for speed, 70B for quality)
- **Smart Resource Management**: Automatic idle timeout, LRU eviction, and configurable instance limits
- **Web Dashboard**: Modern React UI for managing instances, monitoring health, and viewing logs

**🔗 Flexible Integration**
- **OpenAI API Compatible**: Drop-in replacement - route requests to different models by instance name
- **Multi-Backend Support**: Native support for llama.cpp, MLX (Apple Silicon optimized), and vLLM
- **Docker Ready**: Run backends in containers with full GPU support

**🌐 Distributed Deployment**
- **Remote Instances**: Deploy instances on remote hosts
- **Central Management**: Manage everything from a single dashboard with automatic routing  

## Quick Start

1. Install a backend (llama.cpp, MLX, or vLLM) - see [Prerequisites](#prerequisites) below
2. [Download llamactl](#installation) for your platform
3. Run `llamactl` and open http://localhost:8080
4. Create an instance and start inferencing!

## Prerequisites

### Backend Dependencies

**For llama.cpp backend:**
You need `llama-server` from [llama.cpp](https://github.com/ggml-org/llama.cpp) installed:

```bash
# Homebrew (macOS)
brew install llama.cpp

# Or build from source - see llama.cpp docs
# Or use Docker - no local installation required
```

**For MLX backend (macOS only):**
You need MLX-LM installed:

```bash
# Install via pip (requires Python 3.8+)
pip install mlx-lm

# Or in a virtual environment (recommended)
python -m venv mlx-env
source mlx-env/bin/activate
pip install mlx-lm
```

**For vLLM backend:**
You need vLLM installed:

```bash
# Install via pip (requires Python 3.8+, GPU required)
pip install vllm

# Or in a virtual environment (recommended)
python -m venv vllm-env
source vllm-env/bin/activate
pip install vllm

# Or use Docker - no local installation required
```

### Docker Support

llamactl can run backends in Docker containers, eliminating the need for local backend installation:

```yaml
backends:
  llama-cpp:
    docker:
      enabled: true
  vllm:
    docker:
      enabled: true
```

## Installation

### Option 1: Download Binary (Recommended)

```bash
# Linux/macOS - Get latest version and download
LATEST_VERSION=$(curl -s https://api.github.com/repos/lordmathis/llamactl/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -L https://github.com/lordmathis/llamactl/releases/download/${LATEST_VERSION}/llamactl-${LATEST_VERSION}-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz | tar -xz
sudo mv llamactl /usr/local/bin/

# Or download manually from the releases page:
# https://github.com/lordmathis/llamactl/releases/latest

# Windows - Download from releases page
```

### Option 2: Docker (No local backend installation required)

```bash
# Clone repository and build Docker images
git clone https://github.com/lordmathis/llamactl.git
cd llamactl
mkdir -p data/llamacpp data/vllm models

# Build and start llamactl with llama.cpp CUDA backend
docker-compose -f docker/docker-compose.yml up llamactl-llamacpp -d

# Build and start llamactl with vLLM CUDA backend
docker-compose -f docker/docker-compose.yml up llamactl-vllm -d

# Build from source using multi-stage build
docker build -f docker/Dockerfile.source -t llamactl:source .
```

**Note:** Dockerfiles are configured for CUDA. Adapt base images for other platforms (CPU, ROCm, etc.).

### Option 3: Build from Source
Requires Go 1.24+ and Node.js 22+
```bash
git clone https://github.com/lordmathis/llamactl.git
cd llamactl
cd webui && npm ci && npm run build && cd ..
go build -o llamactl ./cmd/server
```

## Usage

1. Open http://localhost:8080
2. Click "Create Instance"
3. Choose backend type (llama.cpp, MLX, or vLLM)
4. Configure your model and options (ports and API keys are auto-assigned)
5. Start the instance and use it with any OpenAI-compatible client

## Configuration

llamactl works out of the box with sensible defaults.

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
    environment: {}               # Environment variables for the backend process
    docker:
      enabled: false
      image: "ghcr.io/ggml-org/llama.cpp:server"
      args: ["run", "--rm", "--network", "host", "--gpus", "all", "-v", "~/.local/share/llamactl/llama.cpp:/root/.cache/llama.cpp"]
      environment: {}             # Environment variables for the container

  vllm:
    command: "vllm"
    args: ["serve"]
    environment: {}               # Environment variables for the backend process
    docker:
      enabled: false
      image: "vllm/vllm-openai:latest"
      args: ["run", "--rm", "--network", "host", "--gpus", "all", "--shm-size", "1g", "-v", "~/.local/share/llamactl/huggingface:/root/.cache/huggingface"]
      environment: {}             # Environment variables for the container

  mlx:
    command: "mlx_lm.server"
    args: []
    environment: {}               # Environment variables for the backend process

data_dir: ~/.local/share/llamactl  # Main data directory (database, instances, logs), default varies by OS

instances:
  port_range: [8000, 9000]                        # Port range for instances
  configs_dir: ~/.local/share/llamactl/instances  # Instance configs directory (platform dependent)
  logs_dir: ~/.local/share/llamactl/logs          # Logs directory (platform dependent)
  auto_create_dirs: true                          # Auto-create data/config/logs dirs if missing
  max_instances: -1                               # Max instances (-1 = unlimited)
  max_running_instances: -1                       # Max running instances (-1 = unlimited)
  enable_lru_eviction: true                       # Enable LRU eviction for idle instances
  default_auto_restart: true                      # Auto-restart new instances by default
  default_max_restarts: 3                         # Max restarts for new instances
  default_restart_delay: 5                        # Restart delay (seconds) for new instances
  default_on_demand_start: true                   # Default on-demand start setting
  on_demand_start_timeout: 120                    # Default on-demand start timeout in seconds
  timeout_check_interval: 5                       # Idle instance timeout check in minutes

database:
  path: ~/.local/share/llamactl/llamactl.db  # Database file path (platform dependent)
  max_open_connections: 25                   # Maximum open database connections
  max_idle_connections: 5                    # Maximum idle database connections
  connection_max_lifetime: 5m                # Connection max lifetime

auth:
  require_inference_auth: true   # Require auth for inference endpoints
  inference_keys: []             # Keys for inference endpoints
  require_management_auth: true  # Require auth for management endpoints
  management_keys: []            # Keys for management endpoints
```

For detailed configuration options including environment variables, file locations, and advanced settings, see the [Configuration Guide](docs/getting-started/configuration.md).

## License

MIT License - see [LICENSE](LICENSE) file.
