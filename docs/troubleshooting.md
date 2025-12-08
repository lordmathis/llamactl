# Troubleshooting

Issues specific to Llamactl deployment and operation.

## Configuration Issues

### Invalid Configuration

**Problem:** Invalid configuration preventing startup

**Solutions:**
1. Use minimal configuration:
   ```yaml
   server:
     host: "0.0.0.0"
     port: 8080
   instances:
     port_range: [8000, 9000]
   ```

2. Check data directory permissions:
   ```bash
   # Ensure data directory is writable (default: ~/.local/share/llamactl)
   mkdir -p ~/.local/share/llamactl/{instances,logs}
   ```

## Instance Management Issues

### Instance Fails to Start

**Problem:** Instance fails to start or immediately stops

**Solutions:**

1. **Check instance logs** to see the actual error:
   ```bash
   curl http://localhost:8080/api/v1/instances/{name}/logs
   # Or check log files directly
   tail -f ~/.local/share/llamactl/logs/{instance-name}.log
   ```

2. **Verify backend is installed:**  
     - **llama.cpp**: Ensure `llama-server` is in PATH
     - **MLX**: Ensure `mlx-lm` Python package is installed
     - **vLLM**: Ensure `vllm` Python package is installed

3. **Check model path and format:**
     - Use absolute paths to model files
     - Verify model format matches backend (GGUF for llama.cpp, etc.)

4. **Verify backend command configuration:**
     - Check that the backend `command` is correctly configured in the global config
     - For virtual environments, specify the full path to the command (e.g., `/path/to/venv/bin/mlx_lm.server`)
     - See the [Configuration Guide](configuration.md) for backend configuration details
     - Test the backend directly (see [Backend-Specific Issues](#backend-specific-issues) below)

### Backend-Specific Issues

**Problem:** Model loading, memory, GPU, or performance issues

Most model-specific issues (memory, GPU configuration, performance tuning) are backend-specific and should be resolved by consulting the respective backend documentation:

**llama.cpp:**
- [llama.cpp GitHub](https://github.com/ggml-org/llama.cpp)
- [llama-server README](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md)

**MLX:**
- [MLX-LM GitHub](https://github.com/ml-explore/mlx-lm)
- [MLX-LM Server Guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/SERVER.md)

**vLLM:**
- [vLLM Documentation](https://docs.vllm.ai/en/stable/)
- [OpenAI Compatible Server](https://docs.vllm.ai/en/stable/serving/openai_compatible_server.html)
- [vllm serve Command](https://docs.vllm.ai/en/stable/cli/serve.html#vllm-serve)

**Testing backends directly:**

Testing your model and configuration directly with the backend helps determine if the issue is with llamactl or the backend itself:

```bash
# llama.cpp
llama-server --model /path/to/model.gguf --port 8081

# MLX
mlx_lm.server --model mlx-community/Mistral-7B-Instruct-v0.3-4bit --port 8081

# vLLM
vllm serve microsoft/DialoGPT-medium --port 8081
```

## API and Network Issues

### CORS Errors

**Problem:** Web UI shows CORS errors in browser console

**Solutions:**
1. **Configure allowed origins:**
   ```yaml
   server:
     allowed_origins:
       - "http://localhost:3000"
       - "https://yourdomain.com"
   ```

## Authentication Issues

**Problem:** API requests failing with authentication errors

**Solutions:**
1. **Disable authentication temporarily:**
   ```yaml
   auth:
     require_management_auth: false
     require_inference_auth: false
   ```

2. **Configure management API keys:**
   ```yaml
   auth:
     management_keys:
       - "your-management-key"
   ```

   For inference API keys, create them via the web UI (Settings → API Keys) after logging in with your management key.

3. **Use correct Authorization header:**
   ```bash
   curl -H "Authorization: Bearer your-api-key" \
     http://localhost:8080/api/v1/instances
   ```

## Remote Node Issues

### Node Configuration

**Problem:** Remote instances not appearing or cannot be managed

**Solutions:**
1. **Verify node configuration:**
   ```yaml
   local_node: "main"  # Must match a key in nodes map
   nodes:
     main:
       address: ""     # Empty for local node
     worker1:
       address: "http://worker1.internal:8080"
       api_key: "secure-key"  # Must match worker1's management key
   ```

2. **Check node name consistency:**
   - `local_node` on each node must match what other nodes call it
   - Node names are case-sensitive

3. **Test remote node connectivity:**
   ```bash
   curl -H "Authorization: Bearer remote-node-key" \
     http://remote-node:8080/api/v1/instances
   ```

## Debugging and Logs

### Viewing Instance Logs

```bash
# Get instance logs via API
curl http://localhost:8080/api/v1/instances/{name}/logs

# Or check log files directly
tail -f ~/.local/share/llamactl/logs/{instance-name}.log
```

### Enable Debug Logging

```bash
export LLAMACTL_LOG_LEVEL=debug
llamactl
```

## Getting Help

When reporting issues, include:

1. **System information:**
   ```bash
   llamactl --version
   ```

2. **Configuration file** (remove sensitive keys)

3. **Relevant log output**

4. **Steps to reproduce the issue**
