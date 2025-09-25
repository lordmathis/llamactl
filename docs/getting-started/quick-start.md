# Quick Start

This guide will help you get Llamactl up and running in just a few minutes.

## Step 1: Start Llamactl

Start the Llamactl server:

```bash
llamactl
```

By default, Llamactl will start on `http://localhost:8080`.

## Step 2: Access the Web UI

Open your web browser and navigate to:

```
http://localhost:8080
```

Login with the management API key. By default it is generated during server startup. Copy it from the terminal output.

You should see the Llamactl web interface.

## Step 3: Create Your First Instance

1. Click the "Add Instance" button
2. Fill in the instance configuration:
   - **Name**: Give your instance a descriptive name
   - **Backend Type**: Choose from llama.cpp, MLX, or vLLM
   - **Model**: Model path or identifier for your chosen backend
   - **Additional Options**: Backend-specific parameters

3. Click "Create Instance"

## Step 4: Start Your Instance

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
  }
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
  }
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
  }
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
curl http://localhost:8080/api/instances

# Create a new llama.cpp instance
curl -X POST http://localhost:8080/api/instances/my-model \
  -H "Content-Type: application/json" \
  -d '{
    "backend_type": "llama_cpp",
    "backend_options": {
      "model": "/path/to/model.gguf"
    }
  }'

# Start an instance
curl -X POST http://localhost:8080/api/instances/my-model/start
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
    api_key="not-needed"  # Llamactl doesn't require API keys by default
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

### List Available Models

Get a list of running instances (models) in OpenAI-compatible format:

```bash
curl http://localhost:8080/v1/models
```

## Next Steps

- Manage instances [Managing Instances](../user-guide/managing-instances.md)
- Explore the [API Reference](../user-guide/api-reference.md)
- Configure advanced settings in the [Configuration](configuration.md) guide
