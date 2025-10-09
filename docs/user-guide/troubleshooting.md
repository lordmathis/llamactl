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

### Model Loading Failures

**Problem:** Instance fails to start with model loading errors

**Common Solutions:**  
- **llama-server not found:** Ensure `llama-server` binary is in PATH  
- **Wrong model format:** Ensure model is in GGUF format  
- **Insufficient memory:** Use smaller model or reduce context size  
- **Path issues:** Use absolute paths to model files  

### Memory Issues

**Problem:** Out of memory errors or system becomes unresponsive

**Solutions:**
1. **Reduce context size:**
   ```json
   {
     "n_ctx": 1024
   }
   ```

2. **Use quantized models:**  
   - Try Q4_K_M instead of higher precision models  
   - Use smaller model variants (7B instead of 13B)  

### GPU Configuration

**Problem:** GPU not being used effectively

**Solutions:**
1. **Configure GPU layers:**
   ```json
   {
     "n_gpu_layers": 35
   }
   ```

### Advanced Instance Issues

**Problem:** Complex model loading, performance, or compatibility issues

Since llamactl uses `llama-server` under the hood, many instance-related issues are actually llama.cpp issues. For advanced troubleshooting:

**Resources:**  
- **llama.cpp Documentation:** [https://github.com/ggml/llama.cpp](https://github.com/ggml/llama.cpp)  
- **llama.cpp Issues:** [https://github.com/ggml/llama.cpp/issues](https://github.com/ggml/llama.cpp/issues)  
- **llama.cpp Discussions:** [https://github.com/ggml/llama.cpp/discussions](https://github.com/ggml/llama.cpp/discussions)  

**Testing directly with llama-server:**  
```bash
# Test your model and parameters directly with llama-server
llama-server --model /path/to/model.gguf --port 8081 --n-gpu-layers 35
```

This helps determine if the issue is with llamactl or with the underlying llama.cpp/llama-server.

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

2. **Configure API keys:**
   ```yaml
   auth:
     management_keys:
       - "your-management-key"
     inference_keys:
       - "your-inference-key"
   ```

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

2. **Test remote node connectivity:**
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
