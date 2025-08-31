# Backends

Llamactl supports multiple backends for running large language models. This guide covers the available backends and their configuration.

## Llama.cpp Backend

The primary backend for Llamactl, providing robust support for GGUF models.

### Features

- **GGUF Support**: Native support for GGUF model format
- **GPU Acceleration**: CUDA, OpenCL, and Metal support
- **Memory Optimization**: Efficient memory usage and mapping
- **Multi-threading**: Configurable CPU thread utilization
- **Quantization**: Support for various quantization levels

### Configuration

```yaml
backends:
  llamacpp:
    binary_path: "/usr/local/bin/llama-server"
    default_options:
      threads: 4
      context_size: 2048
      batch_size: 512
    gpu:
      enabled: true
      layers: 35
```

### Supported Options

| Option | Description | Default |
|--------|-------------|---------|
| `threads` | Number of CPU threads | 4 |
| `context_size` | Context window size | 2048 |
| `batch_size` | Batch size for processing | 512 |
| `gpu_layers` | Layers to offload to GPU | 0 |
| `memory_lock` | Lock model in memory | false |
| `no_mmap` | Disable memory mapping | false |
| `rope_freq_base` | RoPE frequency base | 10000 |
| `rope_freq_scale` | RoPE frequency scale | 1.0 |

### GPU Acceleration

#### CUDA Setup

```bash
# Install CUDA toolkit
sudo apt update
sudo apt install nvidia-cuda-toolkit

# Verify CUDA installation
nvcc --version
nvidia-smi
```

#### Configuration for GPU

```json
{
  "name": "gpu-accelerated",
  "model_path": "/models/llama-2-13b.gguf",
  "port": 8081,
  "options": {
    "gpu_layers": 35,
    "threads": 2,
    "context_size": 4096
  }
}
```

### Performance Tuning

#### Memory Optimization

```yaml
# For limited memory systems
options:
  context_size: 1024
  batch_size: 256
  no_mmap: true
  memory_lock: false

# For high-memory systems
options:
  context_size: 8192
  batch_size: 1024
  memory_lock: true
  no_mmap: false
```

#### CPU Optimization

```yaml
# Match thread count to CPU cores
# For 8-core CPU:
options:
  threads: 6  # Leave 2 cores for system
  
# For high-performance CPUs:
options:
  threads: 16
  batch_size: 1024
```

## Future Backends

Llamactl is designed to support multiple backends. Planned additions:

### vLLM Backend

High-performance inference engine optimized for serving:

- **Features**: Fast inference, batching, streaming
- **Models**: Supports various model formats
- **Scaling**: Horizontal scaling support

### TensorRT-LLM Backend

NVIDIA's optimized inference engine:

- **Features**: Maximum GPU performance
- **Models**: Optimized for NVIDIA GPUs
- **Deployment**: Production-ready inference

### Ollama Backend

Integration with Ollama for easy model management:

- **Features**: Simplified model downloading
- **Models**: Large model library
- **Integration**: Seamless model switching

## Backend Selection

### Automatic Detection

Llamactl can automatically detect the best backend:

```yaml
backends:
  auto_detect: true
  preference_order:
    - "llamacpp"
    - "vllm"
    - "tensorrt"
```

### Manual Selection

Force a specific backend for an instance:

```json
{
  "name": "manual-backend",
  "backend": "llamacpp",
  "model_path": "/models/model.gguf",
  "port": 8081
}
```

## Backend-Specific Features

### Llama.cpp Features

#### Model Formats

- **GGUF**: Primary format, best compatibility
- **GGML**: Legacy format (limited support)

#### Quantization Levels

- `Q2_K`: Smallest size, lower quality
- `Q4_K_M`: Balanced size and quality
- `Q5_K_M`: Higher quality, larger size
- `Q6_K`: Near-original quality
- `Q8_0`: Minimal loss, largest size

#### Advanced Options

```yaml
advanced:
  rope_scaling:
    type: "linear"
    factor: 2.0
  attention:
    flash_attention: true
    grouped_query: true
```

## Monitoring Backend Performance

### Metrics Collection

Monitor backend-specific metrics:

```bash
# Get backend statistics
curl http://localhost:8080/api/instances/my-instance/backend/stats
```

**Response:**
```json
{
  "backend": "llamacpp",
  "version": "b1234",
  "metrics": {
    "tokens_per_second": 15.2,
    "memory_usage": 4294967296,
    "gpu_utilization": 85.5,
    "context_usage": 75.0
  }
}
```

### Performance Optimization

#### Benchmark Different Configurations

```bash
# Test various thread counts
for threads in 2 4 8 16; do
  echo "Testing $threads threads"
  curl -X PUT http://localhost:8080/api/instances/benchmark \
    -d "{\"options\": {\"threads\": $threads}}"
  # Run performance test
done
```

#### Memory Usage Optimization

```bash
# Monitor memory usage
watch -n 1 'curl -s http://localhost:8080/api/instances/my-instance/stats | jq .memory_usage'
```

## Troubleshooting Backends

### Common Llama.cpp Issues

**Model won't load:**
```bash
# Check model file
file /path/to/model.gguf

# Verify format
llama-server --model /path/to/model.gguf --dry-run
```

**GPU not detected:**
```bash
# Check CUDA installation
nvidia-smi

# Verify llama.cpp GPU support
llama-server --help | grep -i gpu
```

**Performance issues:**
```bash
# Check system resources
htop
nvidia-smi

# Verify configuration
curl http://localhost:8080/api/instances/my-instance/config
```

## Custom Backend Development

### Backend Interface

Implement the backend interface for custom backends:

```go
type Backend interface {
    Start(config InstanceConfig) error
    Stop(instance *Instance) error
    Health(instance *Instance) (*HealthStatus, error)
    Stats(instance *Instance) (*Stats, error)
}
```

### Registration

Register your custom backend:

```go
func init() {
    backends.Register("custom", &CustomBackend{})
}
```

## Best Practices

### Production Deployments

1. **Resource allocation**: Plan for peak usage
2. **Backend selection**: Choose based on requirements
3. **Monitoring**: Set up comprehensive monitoring
4. **Fallback**: Configure backup backends

### Development

1. **Rapid iteration**: Use smaller models
2. **Resource monitoring**: Track usage patterns
3. **Configuration testing**: Validate settings
4. **Performance profiling**: Optimize bottlenecks

## Next Steps

- Learn about [Monitoring](monitoring.md) backend performance
- Explore [Troubleshooting](troubleshooting.md) guides
- Set up [Production Monitoring](monitoring.md)
