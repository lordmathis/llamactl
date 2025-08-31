# llamactl

![Build and Release](https://github.com/lordmathis/llamactl/actions/workflows/release.yaml/badge.svg) ![Go Tests](https://github.com/lordmathis/llamactl/actions/workflows/go_test.yaml/badge.svg) ![WebUI Tests](https://github.com/lordmathis/llamactl/actions/workflows/webui_test.yaml/badge.svg)

**Management server and proxy for multiple llama.cpp instances with OpenAI-compatible API routing.**

## Why llamactl?

🚀 **Multiple Model Serving**: Run different models simultaneously (7B for speed, 70B for quality)  
🔗 **OpenAI API Compatible**: Drop-in replacement - route requests by model name  
🌐 **Web Dashboard**: Modern React UI for visual management (unlike CLI-only tools)  
🔐 **API Key Authentication**: Separate keys for management vs inference access  
📊 **Instance Monitoring**: Health checks, auto-restart, log management  
⚡ **Smart Resource Management**: Idle timeout, LRU eviction, and configurable instance limits  
💡 **On-Demand Instance Start**: Automatically launch instances upon receiving OpenAI-compatible API requests  
💾 **State Persistence**: Ensure instances remain intact across server restarts  

![Dashboard Screenshot](docs/images/screenshot.png)

**Choose llamactl if**: You need authentication, health monitoring, auto-restart, and centralized management of multiple llama-server instances  
**Choose Ollama if**: You want the simplest setup with strong community ecosystem and third-party integrations  
**Choose LM Studio if**: You prefer a polished desktop GUI experience with easy model management

## Quick Start

```bash
# 1. Install llama-server (one-time setup)
# See: https://github.com/ggml-org/llama.cpp#quick-start

# 2. Download and run llamactl
LATEST_VERSION=$(curl -s https://api.github.com/repos/lordmathis/llamactl/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -L https://github.com/lordmathis/llamactl/releases/download/${LATEST_VERSION}/llamactl-${LATEST_VERSION}-linux-amd64.tar.gz | tar -xz
sudo mv llamactl /usr/local/bin/

# 3. Start the server
llamactl
# Access dashboard at http://localhost:8080
```

## Usage

### Create and manage instances via web dashboard:
1. Open http://localhost:8080
2. Click "Create Instance"
3. Set model path and GPU layers
4. Start or stop the instance

### Or use the REST API:
```bash
# Create instance
curl -X POST localhost:8080/api/v1/instances/my-7b-model \
  -H "Authorization: Bearer your-key" \
  -d '{"model": "/path/to/model.gguf", "gpu_layers": 32}'

# Use with OpenAI SDK
curl -X POST localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-key" \
  -d '{"model": "my-7b-model", "messages": [{"role": "user", "content": "Hello!"}]}'
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

### Option 2: Build from Source
Requires Go 1.24+ and Node.js 22+
```bash
git clone https://github.com/lordmathis/llamactl.git
cd llamactl
cd webui && npm ci && npm run build && cd ..
go build -o llamactl ./cmd/server
```

## Prerequisites

You need `llama-server` from [llama.cpp](https://github.com/ggml-org/llama.cpp) installed:

```bash
# Quick install methods:
# Homebrew (macOS)
brew install llama.cpp

# Or build from source - see llama.cpp docs
```

## Configuration

llamactl works out of the box with sensible defaults.

```yaml
server:
  host: "0.0.0.0"                # Server host to bind to
  port: 8080                     # Server port to bind to
  allowed_origins: ["*"]         # Allowed CORS origins (default: all)
  enable_swagger: false          # Enable Swagger UI for API docs

instances:
  port_range: [8000, 9000]       # Port range for instances
  data_dir: ~/.local/share/llamactl         # Data directory (platform-specific, see below)
  configs_dir: ~/.local/share/llamactl/instances  # Instance configs directory
  logs_dir: ~/.local/share/llamactl/logs    # Logs directory
  auto_create_dirs: true         # Auto-create data/config/logs dirs if missing
  max_instances: -1              # Max instances (-1 = unlimited)
  max_running_instances: -1      # Max running instances (-1 = unlimited)
  enable_lru_eviction: true      # Enable LRU eviction for idle instances
  llama_executable: llama-server # Path to llama-server executable
  default_auto_restart: true     # Auto-restart new instances by default
  default_max_restarts: 3        # Max restarts for new instances
  default_restart_delay: 5       # Restart delay (seconds) for new instances
  default_on_demand_start: true  # Default on-demand start setting
  on_demand_start_timeout: 120   # Default on-demand start timeout in seconds
  timeout_check_interval: 5      # Idle instance timeout check in minutes

auth:
  require_inference_auth: true   # Require auth for inference endpoints
  inference_keys: []             # Keys for inference endpoints
  require_management_auth: true  # Require auth for management endpoints
  management_keys: []            # Keys for management endpoints
```

For detailed configuration options including environment variables, file locations, and advanced settings, see the [Configuration Guide](docs/getting-started/configuration.md).

## License

MIT License - see [LICENSE](LICENSE) file.
