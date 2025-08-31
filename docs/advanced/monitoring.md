# Monitoring

Comprehensive monitoring setup for Llamactl in production environments.

## Overview

Effective monitoring of Llamactl involves tracking:

- Instance health and performance
- System resource usage
- API response times
- Error rates and alerts

## Built-in Monitoring

### Health Checks

Llamactl provides built-in health monitoring:

```bash
# Check overall system health
curl http://localhost:8080/api/system/health

# Check specific instance health
curl http://localhost:8080/api/instances/{name}/health
```

### Metrics Endpoint

Access Prometheus-compatible metrics:

```bash
curl http://localhost:8080/metrics
```

**Available Metrics:**
- `llamactl_instances_total`: Total number of instances
- `llamactl_instances_running`: Number of running instances
- `llamactl_instance_memory_bytes`: Instance memory usage
- `llamactl_instance_cpu_percent`: Instance CPU usage
- `llamactl_api_requests_total`: Total API requests
- `llamactl_api_request_duration_seconds`: API response times

## Prometheus Integration

### Configuration

Add Llamactl as a Prometheus target:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'llamactl'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Custom Metrics

Enable additional metrics in Llamactl:

```yaml
# config.yaml
monitoring:
  enabled: true
  prometheus:
    enabled: true
    path: "/metrics"
  metrics:
    - instance_stats
    - api_performance
    - system_resources
```

## Grafana Dashboards

### Llamactl Dashboard

Import the official Grafana dashboard:

1. Download dashboard JSON from releases
2. Import into Grafana
3. Configure Prometheus data source

### Key Panels

**Instance Overview:**
- Instance count and status
- Resource usage per instance
- Health status indicators

**Performance Metrics:**
- API response times
- Tokens per second
- Memory usage trends

**System Resources:**
- CPU and memory utilization
- Disk I/O and network usage
- GPU utilization (if applicable)

### Custom Queries

**Instance Uptime:**
```promql
(time() - llamactl_instance_start_time_seconds) / 3600
```

**Memory Usage Percentage:**
```promql
(llamactl_instance_memory_bytes / llamactl_system_memory_total_bytes) * 100
```

**API Error Rate:**
```promql
rate(llamactl_api_requests_total{status=~"4.."}[5m]) / rate(llamactl_api_requests_total[5m]) * 100
```

## Alerting

### Prometheus Alerts

Configure alerts for critical conditions:

```yaml
# alerts.yml
groups:
  - name: llamactl
    rules:
      - alert: InstanceDown
        expr: llamactl_instance_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Llamactl instance {{ $labels.instance_name }} is down"
          
      - alert: HighMemoryUsage
        expr: llamactl_instance_memory_percent > 90
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.instance_name }}"
          
      - alert: APIHighLatency
        expr: histogram_quantile(0.95, rate(llamactl_api_request_duration_seconds_bucket[5m])) > 2
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High API latency detected"
```

### Notification Channels

Configure alert notifications:

**Slack Integration:**
```yaml
# alertmanager.yml
route:
  group_by: ['alertname']
  receiver: 'slack'

receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#alerts'
        title: 'Llamactl Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

## Log Management

### Centralized Logging

Configure log aggregation:

```yaml
# config.yaml
logging:
  level: "info"
  output: "json"
  destinations:
    - type: "file"
      path: "/var/log/llamactl/app.log"
    - type: "syslog"
      facility: "local0"
    - type: "elasticsearch"
      url: "http://elasticsearch:9200"
```

### Log Analysis

Use ELK stack for log analysis:

**Elasticsearch Index Template:**
```json
{
  "index_patterns": ["llamactl-*"],
  "mappings": {
    "properties": {
      "timestamp": {"type": "date"},
      "level": {"type": "keyword"},
      "message": {"type": "text"},
      "instance": {"type": "keyword"},
      "component": {"type": "keyword"}
    }
  }
}
```

**Kibana Visualizations:**
- Log volume over time
- Error rate by instance
- Performance trends
- Resource usage patterns

## Application Performance Monitoring

### OpenTelemetry Integration

Enable distributed tracing:

```yaml
# config.yaml
telemetry:
  enabled: true
  otlp:
    endpoint: "http://jaeger:14268/api/traces"
  sampling_rate: 0.1
```

### Custom Spans

Add custom tracing to track operations:

```go
ctx, span := tracer.Start(ctx, "instance.start")
defer span.End()

// Track instance startup time
span.SetAttributes(
    attribute.String("instance.name", name),
    attribute.String("model.path", modelPath),
)
```

## Health Check Configuration

### Readiness Probes

Configure Kubernetes readiness probes:

```yaml
readinessProbe:
  httpGet:
    path: /api/health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
```

### Liveness Probes

Configure liveness probes:

```yaml
livenessProbe:
  httpGet:
    path: /api/health/live
    port: 8080
  initialDelaySeconds: 60
  periodSeconds: 30
```

### Custom Health Checks

Implement custom health checks:

```go
func (h *HealthHandler) CustomCheck(ctx context.Context) error {
    // Check database connectivity
    if err := h.db.Ping(); err != nil {
        return fmt.Errorf("database unreachable: %w", err)
    }
    
    // Check instance responsiveness
    for _, instance := range h.instances {
        if !instance.IsHealthy() {
            return fmt.Errorf("instance %s unhealthy", instance.Name)
        }
    }
    
    return nil
}
```

## Performance Profiling

### pprof Integration

Enable Go profiling:

```yaml
# config.yaml
debug:
  pprof_enabled: true
  pprof_port: 6060
```

Access profiling endpoints:
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Continuous Profiling

Set up continuous profiling with Pyroscope:

```yaml
# config.yaml
profiling:
  enabled: true
  pyroscope:
    server_address: "http://pyroscope:4040"
    application_name: "llamactl"
```

## Security Monitoring

### Audit Logging

Enable security audit logs:

```yaml
# config.yaml
audit:
  enabled: true
  log_file: "/var/log/llamactl/audit.log"
  events:
    - "auth.login"
    - "auth.logout"
    - "instance.create"
    - "instance.delete"
    - "config.update"
```

### Rate Limiting Monitoring

Track rate limiting metrics:

```bash
# Monitor rate limit hits
curl http://localhost:8080/metrics | grep rate_limit
```

## Troubleshooting Monitoring

### Common Issues

**Metrics not appearing:**
1. Check Prometheus configuration
2. Verify network connectivity
3. Review Llamactl logs for errors

**High memory usage:**
1. Check for memory leaks in profiles
2. Monitor garbage collection metrics
3. Review instance configurations

**Alert fatigue:**
1. Tune alert thresholds
2. Implement alert severity levels
3. Use alert routing and suppression

### Debug Tools

**Monitoring health:**
```bash
# Check monitoring endpoints
curl -v http://localhost:8080/metrics
curl -v http://localhost:8080/api/health

# Review logs
tail -f /var/log/llamactl/app.log
```

## Best Practices

### Production Monitoring

1. **Comprehensive coverage**: Monitor all critical components
2. **Appropriate alerting**: Balance sensitivity and noise
3. **Regular review**: Analyze trends and patterns
4. **Documentation**: Maintain runbooks for alerts

### Performance Optimization

1. **Baseline establishment**: Know normal operating parameters
2. **Trend analysis**: Identify performance degradation early
3. **Capacity planning**: Monitor resource growth trends
4. **Optimization cycles**: Regular performance tuning

## Next Steps

- Set up [Troubleshooting](troubleshooting.md) procedures
- Learn about [Backend optimization](backends.md)
- Configure [Production deployment](../development/building.md)
