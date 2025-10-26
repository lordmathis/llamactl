# Quick Start

This guide will help you get Llamactl up and running in just a few minutes.

**Before you begin:** Ensure you have at least one backend installed (llama.cpp, MLX, or vLLM). See the [Installation Guide](installation.md#prerequisites) for backend setup.

## Core Concepts

Before you start, let's clarify a few key terms:

- **Instance**: A running backend server that serves a specific model. Each instance has a unique name and runs independently.
- **Backend**: The inference engine that actually runs the model (llama.cpp, MLX, or vLLM). You need at least one backend installed before creating instances.
- **Node**: In multi-machine setups, a node represents one machine. Most users will just use the default "main" node for single-machine deployments.
- **Proxy Architecture**: Llamactl acts as a proxy in front of your instances. You make requests to llamactl (e.g., `http://localhost:8080/v1/chat/completions`), and it routes them to the appropriate backend instance. This means you don't need to track individual instance ports or endpoints.

## Authentication

Llamactl uses two types of API keys:

- **Management API Key**: Used to authenticate with the Llamactl management API (creating, starting, stopping instances).
- **Inference API Key**: Used to authenticate requests to the OpenAI-compatible endpoints (`/v1/chat/completions`, `/v1/completions`, etc.).

By default, authentication is required. If you don't configure these keys in your configuration file, llamactl will auto-generate them and print them to the terminal on startup. You can also configure custom keys or disable authentication entirely in the [Configuration](configuration.md) guide.

## Start Llamactl

Start the Llamactl server:

```bash
llamactl
```

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  MANAGEMENT AUTHENTICATION REQUIRED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔑  Generated Management API Key:

    sk-management-...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  INFERENCE AUTHENTICATION REQUIRED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔑  Generated Inference API Key:

    sk-inference-...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  IMPORTANT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• These keys are auto-generated and will change on restart
• For production, add explicit keys to your configuration
• Copy these keys before they disappear from the terminal
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Llamactl server listening on 0.0.0.0:8080
```

Copy the **Management** and **Inference** API Keys from the terminal - you'll need them to access the web UI and make inference requests.

By default, Llamactl will start on `http://localhost:8080`.

## Access the Web UI

Open your web browser and navigate to:

```
http://localhost:8080
```

Login with the management API key from the terminal output.

You should see the Llamactl web interface.

## Create Your First Instance

1. Click the "Add Instance" button
2. Fill in the instance configuration:
     - **Name**: Give your instance a descriptive name
     - **Node**: Select which node to deploy the instance to (defaults to "main" for single-node setups)
     - **Backend Type**: Choose from llama.cpp, MLX, or vLLM
     - **Model**: Model path or huggingface repo
     - **Additional Options**: Backend-specific parameters

    !!! tip "Auto-Assignment"
        Llamactl automatically assigns ports from the configured port range (default: 8000-9000) and generates API keys if authentication is enabled. You typically don't need to manually specify these values.

    !!! note "Remote Node Deployment"
        If you have configured remote nodes in your configuration file, you can select which node to deploy the instance to. This allows you to distribute instances across multiple machines. See the [Configuration](configuration.md#remote-node-configuration) guide for details on setting up remote nodes.

3. Click "Create Instance"

## Start Your Instance

Once created, you can:

- **Start** the instance by clicking the start button
- **Monitor** its status in real-time
- **View logs** by clicking the logs button
- **Stop** the instance when needed

## Example Configurations

Here are basic example configurations for each backend:

**llama.cpp backend:**
```json
{
  "name": "llama2-7b",
  "backend_type": "llama_cpp",
  "backend_options": {
    "model": "/path/to/llama-2-7b-chat.gguf",
    "threads": 4,
    "ctx_size": 2048,
    "gpu_layers": 32
  },
  "nodes": ["main"]
}
```

**MLX backend (macOS only):**
```json
{
  "name": "mistral-mlx",
  "backend_type": "mlx_lm",
  "backend_options": {
    "model": "mlx-community/Mistral-7B-Instruct-v0.3-4bit",
    "temp": 0.7,
    "max_tokens": 2048
  },
  "nodes": ["main"]
}
```

**vLLM backend:**
```json
{
  "name": "dialogpt-vllm",
  "backend_type": "vllm",
  "backend_options": {
    "model": "microsoft/DialoGPT-medium",
    "tensor_parallel_size": 2,
    "gpu_memory_utilization": 0.9
  },
  "nodes": ["main"]
}
```

**Remote node deployment example:**
```json
{
  "name": "distributed-model",
  "backend_type": "llama_cpp",
  "backend_options": {
    "model": "/path/to/model.gguf",
    "gpu_layers": 32
  },
  "nodes": ["worker1"]
}
```

## Docker Support

Llamactl can run backends in Docker containers. To enable Docker for a backend, add a `docker` section to that backend in your YAML configuration file (e.g. `config.yaml`) as shown below:

```yaml
backends:
  vllm:
    command: "vllm"
    args: ["serve"]
    docker:
      enabled: true
      image: "vllm/vllm-openai:latest"
      args: ["run", "--rm", "--network", "host", "--gpus", "all", "--shm-size", "1g"]
```

## Using the API

You can also manage instances via the REST API:

```bash
# List all instances
curl http://localhost:8080/api/v1/instances

# Create a new llama.cpp instance
curl -X POST http://localhost:8080/api/v1/instances/my-model \
  -H "Content-Type: application/json" \
  -d '{
    "backend_type": "llama_cpp",
    "backend_options": {
      "model": "/path/to/model.gguf"
    }
  }'

# Start an instance
curl -X POST http://localhost:8080/api/v1/instances/my-model/start
```

## OpenAI Compatible API

Llamactl provides OpenAI-compatible endpoints, making it easy to integrate with existing OpenAI client libraries and tools.

### Chat Completions

Once you have an instance running, you can use it with the OpenAI-compatible chat completions endpoint:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-model",
    "messages": [
      {
        "role": "user",
        "content": "Hello! Can you help me write a Python function?"
      }
    ],
    "max_tokens": 150,
    "temperature": 0.7
  }'
```

### Using with Python OpenAI Client

You can also use the official OpenAI Python client:

```python
from openai import OpenAI

# Point the client to your Llamactl server
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-inference-api-key"  # Use the inference API key from terminal or config
)

# Create a chat completion
response = client.chat.completions.create(
    model="my-model",  # Use the name of your instance
    messages=[
        {"role": "user", "content": "Explain quantum computing in simple terms"}
    ],
    max_tokens=200,
    temperature=0.7
)

print(response.choices[0].message.content)
```

!!! note "API Key"
    If you disabled authentication in your config, you can use any value for `api_key` (e.g., `"not-needed"`). Otherwise, use the inference API key shown in the terminal output on startup.

### List Available Models

Get a list of running instances (models) in OpenAI-compatible format:

```bash
curl http://localhost:8080/v1/models
```

## Next Steps

- Manage instances [Managing Instances](managing-instances.md)
- Explore the [API Reference](api-reference.md)
- Configure advanced settings in the [Configuration](configuration.md) guide
