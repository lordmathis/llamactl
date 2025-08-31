# Troubleshooting

Common issues and solutions for Llamactl deployment and operation.

## Installation Issues

### Binary Not Found

**Problem:** `llamactl: command not found`

**Solutions:**
1. Verify the binary is in your PATH:
   ```bash
   echo $PATH
   which llamactl
   ```

2. Add to PATH or use full path:
   ```bash
   export PATH=$PATH:/path/to/llamactl
   # or
   /full/path/to/llamactl
   ```

3. Check binary permissions:
   ```bash
   chmod +x llamactl
   ```

### Permission Denied

**Problem:** Permission errors when starting Llamactl

**Solutions:**
1. Check file permissions:
   ```bash
   ls -la llamactl
   chmod +x llamactl
   ```

2. Verify directory permissions:
   ```bash
   # Check models directory
   ls -la /path/to/models/
   
   # Check logs directory
   sudo mkdir -p /var/log/llamactl
   sudo chown $USER:$USER /var/log/llamactl
   ```

3. Run with appropriate user:
   ```bash
   # Don't run as root unless necessary
   sudo -u llamactl ./llamactl
   ```

## Startup Issues

### Port Already in Use

**Problem:** `bind: address already in use`

**Solutions:**
1. Find process using the port:
   ```bash
   sudo netstat -tulpn | grep :8080
   # or
   sudo lsof -i :8080
   ```

2. Kill the conflicting process:
   ```bash
   sudo kill -9 <PID>
   ```

3. Use a different port:
   ```bash
   llamactl --port 8081
   ```

### Configuration Errors

**Problem:** Invalid configuration preventing startup

**Solutions:**
1. Validate configuration file:
   ```bash
   llamactl --config /path/to/config.yaml --validate
   ```

2. Check YAML syntax:
   ```bash
   yamllint config.yaml
   ```

3. Use minimal configuration:
   ```yaml
   server:
     host: "localhost"
     port: 8080
   ```

## Instance Management Issues

### Model Loading Failures

**Problem:** Instance fails to start with model loading errors

**Diagnostic Steps:**
1. Check model file exists:
   ```bash
   ls -la /path/to/model.gguf
   file /path/to/model.gguf
   ```

2. Verify model format:
   ```bash
   # Check if it's a valid GGUF file
   hexdump -C /path/to/model.gguf | head -5
   ```

3. Test with llama.cpp directly:
   ```bash
   llama-server --model /path/to/model.gguf --port 8081
   ```

**Common Solutions:**
- **Corrupted model:** Re-download the model file
- **Wrong format:** Ensure model is in GGUF format
- **Insufficient memory:** Reduce context size or use smaller model
- **Path issues:** Use absolute paths, check file permissions

### Memory Issues

**Problem:** Out of memory errors or system becomes unresponsive

**Diagnostic Steps:**
1. Check system memory:
   ```bash
   free -h
   cat /proc/meminfo
   ```

2. Monitor memory usage:
   ```bash
   top -p $(pgrep llamactl)
   ```

3. Check instance memory requirements:
   ```bash
   curl http://localhost:8080/api/instances/{name}/stats
   ```

**Solutions:**
1. **Reduce context size:**
   ```json
   {
     "options": {
       "context_size": 1024
     }
   }
   ```

2. **Enable memory mapping:**
   ```json
   {
     "options": {
       "no_mmap": false
     }
   }
   ```

3. **Use quantized models:**
   - Try Q4_K_M instead of higher precision models
   - Use smaller model variants (7B instead of 13B)

### GPU Issues

**Problem:** GPU not detected or not being used

**Diagnostic Steps:**
1. Check GPU availability:
   ```bash
   nvidia-smi
   ```

2. Verify CUDA installation:
   ```bash
   nvcc --version
   ```

3. Check llama.cpp GPU support:
   ```bash
   llama-server --help | grep -i gpu
   ```

**Solutions:**
1. **Install CUDA drivers:**
   ```bash
   sudo apt update
   sudo apt install nvidia-driver-470 nvidia-cuda-toolkit
   ```

2. **Rebuild llama.cpp with GPU support:**
   ```bash
   cmake -DLLAMA_CUBLAS=ON ..
   make
   ```

3. **Configure GPU layers:**
   ```json
   {
     "options": {
       "gpu_layers": 35
     }
   }
   ```

## Performance Issues

### Slow Response Times

**Problem:** API responses are slow or timeouts occur

**Diagnostic Steps:**
1. Check API response times:
   ```bash
   time curl http://localhost:8080/api/instances
   ```

2. Monitor system resources:
   ```bash
   htop
   iotop
   ```

3. Check instance logs:
   ```bash
   curl http://localhost:8080/api/instances/{name}/logs
   ```

**Solutions:**
1. **Optimize thread count:**
   ```json
   {
     "options": {
       "threads": 6
     }
   }
   ```

2. **Adjust batch size:**
   ```json
   {
     "options": {
       "batch_size": 512
     }
   }
   ```

3. **Enable GPU acceleration:**
   ```json
   {
     "options": {
       "gpu_layers": 35
     }
   }
   ```

### High CPU Usage

**Problem:** Llamactl consuming excessive CPU

**Diagnostic Steps:**
1. Identify CPU-intensive processes:
   ```bash
   top -p $(pgrep -f llamactl)
   ```

2. Check thread allocation:
   ```bash
   curl http://localhost:8080/api/instances/{name}/config
   ```

**Solutions:**
1. **Reduce thread count:**
   ```json
   {
     "options": {
       "threads": 4
     }
   }
   ```

2. **Limit concurrent instances:**
   ```yaml
   limits:
     max_instances: 3
   ```

## Network Issues

### Connection Refused

**Problem:** Cannot connect to Llamactl web interface

**Diagnostic Steps:**
1. Check if service is running:
   ```bash
   ps aux | grep llamactl
   ```

2. Verify port binding:
   ```bash
   netstat -tulpn | grep :8080
   ```

3. Test local connectivity:
   ```bash
   curl http://localhost:8080/api/health
   ```

**Solutions:**
1. **Check firewall settings:**
   ```bash
   sudo ufw status
   sudo ufw allow 8080
   ```

2. **Bind to correct interface:**
   ```yaml
   server:
     host: "0.0.0.0"  # Instead of "localhost"
     port: 8080
   ```

### CORS Errors

**Problem:** Web UI shows CORS errors in browser console

**Solutions:**
1. **Enable CORS in configuration:**
   ```yaml
   server:
     cors_enabled: true
     cors_origins:
       - "http://localhost:3000"
       - "https://yourdomain.com"
   ```

2. **Use reverse proxy:**
   ```nginx
   server {
       listen 80;
       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

## Database Issues

### Startup Database Errors

**Problem:** Database connection failures on startup

**Diagnostic Steps:**
1. Check database service:
   ```bash
   systemctl status postgresql
   # or
   systemctl status mysql
   ```

2. Test database connectivity:
   ```bash
   psql -h localhost -U llamactl -d llamactl
   ```

**Solutions:**
1. **Start database service:**
   ```bash
   sudo systemctl start postgresql
   sudo systemctl enable postgresql
   ```

2. **Create database and user:**
   ```sql
   CREATE DATABASE llamactl;
   CREATE USER llamactl WITH PASSWORD 'password';
   GRANT ALL PRIVILEGES ON DATABASE llamactl TO llamactl;
   ```

## Web UI Issues

### Blank Page or Loading Issues

**Problem:** Web UI doesn't load or shows blank page

**Diagnostic Steps:**
1. Check browser console for errors (F12)
2. Verify API connectivity:
   ```bash
   curl http://localhost:8080/api/system/status
   ```

3. Check static file serving:
   ```bash
   curl http://localhost:8080/
   ```

**Solutions:**
1. **Clear browser cache**
2. **Try different browser**
3. **Check for JavaScript errors in console**
4. **Verify API endpoint accessibility**

### Authentication Issues

**Problem:** Unable to login or authentication failures

**Diagnostic Steps:**
1. Check authentication configuration:
   ```bash
   curl http://localhost:8080/api/config | jq .auth
   ```

2. Verify user credentials:
   ```bash
   # Test login endpoint
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"password"}'
   ```

**Solutions:**
1. **Reset admin password:**
   ```bash
   llamactl --reset-admin-password
   ```

2. **Disable authentication temporarily:**
   ```yaml
   auth:
     enabled: false
   ```

## Log Analysis

### Enable Debug Logging

For detailed troubleshooting, enable debug logging:

```yaml
logging:
  level: "debug"
  output: "/var/log/llamactl/debug.log"
```

### Key Log Patterns

Look for these patterns in logs:

**Startup issues:**
```
ERRO Failed to start server
ERRO Database connection failed
ERRO Port binding failed
```

**Instance issues:**
```
ERRO Failed to start instance
ERRO Model loading failed
ERRO Process crashed
```

**Performance issues:**
```
WARN High memory usage detected
WARN Request timeout
WARN Resource limit exceeded
```

## Getting Help

### Collecting Information

When seeking help, provide:

1. **System information:**
   ```bash
   uname -a
   llamactl --version
   ```

2. **Configuration:**
   ```bash
   llamactl --config-dump
   ```

3. **Logs:**
   ```bash
   tail -100 /var/log/llamactl/app.log
   ```

4. **Error details:**
   - Exact error messages
   - Steps to reproduce
   - Environment details

### Support Channels

- **GitHub Issues:** Report bugs and feature requests
- **Documentation:** Check this documentation first
- **Community:** Join discussions in GitHub Discussions

## Preventive Measures

### Health Monitoring

Set up monitoring to catch issues early:

```bash
# Regular health checks
*/5 * * * * curl -f http://localhost:8080/api/health || alert
```

### Resource Monitoring

Monitor system resources:

```bash
# Disk space monitoring
df -h /var/log/llamactl/
df -h /path/to/models/

# Memory monitoring
free -h
```

### Backup Configuration

Regular configuration backups:

```bash
# Backup configuration
cp ~/.llamactl/config.yaml ~/.llamactl/config.yaml.backup

# Backup instance configurations
curl http://localhost:8080/api/instances > instances-backup.json
```

## Next Steps

- Set up [Monitoring](monitoring.md) to prevent issues
- Learn about [Advanced Configuration](backends.md)
- Review [Best Practices](../development/contributing.md)
