# Llamactl Documentation

Welcome to the Llamactl documentation! **Management server and proxy for multiple llama.cpp instances with OpenAI-compatible API routing.**

![Dashboard Screenshot](images/screenshot.png)

## What is Llamactl?

Llamactl is designed to simplify the deployment and management of llama-server instances. It provides a modern solution for running multiple large language models with centralized management.

## Why llamactl?

🚀 **Multiple Model Serving**: Run different models simultaneously (7B for speed, 70B for quality)  
🔗 **OpenAI API Compatible**: Drop-in replacement - route requests by model name  
🌐 **Web Dashboard**: Modern React UI for visual management (unlike CLI-only tools)  
🔐 **API Key Authentication**: Separate keys for management vs inference access  
📊 **Instance Monitoring**: Health checks, auto-restart, log management  
⚡ **Smart Resource Management**: Idle timeout, LRU eviction, and configurable instance limits  
💡 **On-Demand Instance Start**: Automatically launch instances upon receiving OpenAI-compatible API requests  
💾 **State Persistence**: Ensure instances remain intact across server restarts  

**Choose llamactl if**: You need authentication, health monitoring, auto-restart, and centralized management of multiple llama-server instances  
**Choose Ollama if**: You want the simplest setup with strong community ecosystem and third-party integrations  
**Choose LM Studio if**: You prefer a polished desktop GUI experience with easy model management

## Key Features

- 🚀 **Easy Setup**: Quick installation and configuration
- 🌐 **Web Interface**: Intuitive web UI for model management
- 🔧 **REST API**: Full API access for automation
- 📊 **Monitoring**: Real-time health and status monitoring
- 🔒 **Security**: Authentication and access control
- 📱 **Responsive**: Works on desktop and mobile devices

## Quick Links

- [Installation Guide](getting-started/installation.md) - Get Llamactl up and running
- [Configuration Guide](getting-started/configuration.md) - Detailed configuration options
- [Quick Start](getting-started/quick-start.md) - Your first steps with Llamactl
- [Web UI Guide](user-guide/web-ui.md) - Learn to use the web interface
- [Managing Instances](user-guide/managing-instances.md) - Instance lifecycle management
- [API Reference](user-guide/api-reference.md) - Complete API documentation
- [Monitoring](advanced/monitoring.md) - Health checks and monitoring
- [Backends](advanced/backends.md) - Backend configuration options

## Getting Help

If you need help or have questions:

- Check the [Troubleshooting](advanced/troubleshooting.md) guide
- Visit the [GitHub repository](https://github.com/lordmathis/llamactl)
- Review the [Configuration Guide](getting-started/configuration.md) for advanced settings

## License

MIT License - see the [LICENSE](https://github.com/lordmathis/llamactl/blob/main/LICENSE) file.
