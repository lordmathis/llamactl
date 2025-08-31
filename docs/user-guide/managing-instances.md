# Managing Instances

Learn how to effectively manage your Llama.cpp instances with Llamactl.

## Instance Lifecycle

### Creating Instances

Instances can be created through the Web UI or API:

#### Via Web UI
1. Click "Add Instance" button
2. Fill in the configuration form
3. Click "Create"

#### Via API
```bash
curl -X POST http://localhost:8080/api/instances \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-instance",
    "model_path": "/path/to/model.gguf",
    "port": 8081
  }'
```

### Starting and Stopping

#### Start an Instance
```bash
# Via API
curl -X POST http://localhost:8080/api/instances/{name}/start

# The instance will begin loading the model
```

#### Stop an Instance
```bash
# Via API
curl -X POST http://localhost:8080/api/instances/{name}/stop

# Graceful shutdown with configurable timeout
```

### Monitoring Status

Check instance status in real-time:

```bash
# Get instance details
curl http://localhost:8080/api/instances/{name}

# Get health status
curl http://localhost:8080/api/instances/{name}/health
```

## Instance States

Instances can be in one of several states:

- **Stopped**: Instance is not running
- **Starting**: Instance is initializing and loading the model
- **Running**: Instance is active and ready to serve requests
- **Stopping**: Instance is shutting down gracefully
- **Error**: Instance encountered an error

## Configuration Management

### Updating Instance Configuration

Modify instance settings:

```bash
curl -X PUT http://localhost:8080/api/instances/{name} \
  -H "Content-Type: application/json" \
  -d '{
    "options": {
      "threads": 8,
      "context_size": 4096
    }
  }'
```

!!! note
    Configuration changes require restarting the instance to take effect.

### Viewing Configuration

```bash
# Get current configuration
curl http://localhost:8080/api/instances/{name}/config
```

## Resource Management

### Memory Usage

Monitor memory consumption:

```bash
# Get resource usage
curl http://localhost:8080/api/instances/{name}/stats
```

### CPU and GPU Usage

Track performance metrics:

- CPU thread utilization
- GPU memory usage (if applicable)
- Request processing times

## Troubleshooting Common Issues

### Instance Won't Start

1. **Check model path**: Ensure the model file exists and is readable
2. **Port conflicts**: Verify the port isn't already in use
3. **Resource limits**: Check available memory and CPU
4. **Permissions**: Ensure proper file system permissions

### Performance Issues

1. **Adjust thread count**: Match to your CPU cores
2. **Optimize context size**: Balance memory usage and capability
3. **GPU offloading**: Use `gpu_layers` for GPU acceleration
4. **Batch size tuning**: Optimize for your workload

### Memory Problems

1. **Reduce context size**: Lower memory requirements
2. **Disable memory mapping**: Use `no_mmap` option
3. **Enable memory locking**: Use `memory_lock` for performance
4. **Monitor system resources**: Check available RAM

## Best Practices

### Production Deployments

1. **Resource allocation**: Plan memory and CPU requirements
2. **Health monitoring**: Set up regular health checks
3. **Graceful shutdowns**: Use proper stop procedures
4. **Backup configurations**: Save instance configurations
5. **Log management**: Configure appropriate logging levels

### Development Environments

1. **Resource sharing**: Use smaller models for development
2. **Quick iterations**: Optimize for fast startup times
3. **Debug logging**: Enable detailed logging for troubleshooting

## Batch Operations

### Managing Multiple Instances

```bash
# Start all instances
curl -X POST http://localhost:8080/api/instances/start-all

# Stop all instances
curl -X POST http://localhost:8080/api/instances/stop-all

# Get status of all instances
curl http://localhost:8080/api/instances
```

## Next Steps

- Learn about the [Web UI](web-ui.md) interface
- Explore the complete [API Reference](api-reference.md)
- Set up [Monitoring](../advanced/monitoring.md) for production use
