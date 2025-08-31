# Web UI Guide

The Llamactl Web UI provides an intuitive interface for managing your Llama.cpp instances.

## Overview

The web interface is accessible at `http://localhost:8080` (or your configured host/port) and provides:

- Instance management dashboard
- Real-time status monitoring
- Configuration management
- Log viewing
- System information

## Dashboard

### Instance Cards

Each instance is displayed as a card showing:

- **Instance name** and status indicator
- **Model information** (name, size)
- **Current state** (stopped, starting, running, error)
- **Resource usage** (memory, CPU)
- **Action buttons** (start, stop, configure, logs)

### Status Indicators

- 🟢 **Green**: Instance is running and healthy
- 🟡 **Yellow**: Instance is starting or stopping
- 🔴 **Red**: Instance has encountered an error
- ⚪ **Gray**: Instance is stopped

## Creating Instances

### Add Instance Dialog

1. Click the **"Add Instance"** button
2. Fill in the required fields:
   - **Name**: Unique identifier for your instance
   - **Model Path**: Full path to your GGUF model file
   - **Port**: Port number for the instance

3. Configure optional settings:
   - **Threads**: Number of CPU threads
   - **Context Size**: Context window size
   - **GPU Layers**: Layers to offload to GPU
   - **Additional Options**: Advanced Llama.cpp parameters

4. Click **"Create"** to save the instance

### Model Path Helper

Use the file browser to select model files:

- Navigate to your models directory
- Select the `.gguf` file
- Path is automatically filled in the form

## Managing Instances

### Starting Instances

1. Click the **"Start"** button on an instance card
2. Watch the status change to "Starting"
3. Monitor progress in the logs
4. Instance becomes "Running" when ready

### Stopping Instances

1. Click the **"Stop"** button
2. Instance gracefully shuts down
3. Status changes to "Stopped"

### Viewing Logs

1. Click the **"Logs"** button on any instance
2. Real-time log viewer opens
3. Filter by log level (Debug, Info, Warning, Error)
4. Search through log entries
5. Download logs for offline analysis

## Configuration Management

### Editing Instance Settings

1. Click the **"Configure"** button
2. Modify settings in the configuration dialog
3. Changes require instance restart to take effect
4. Click **"Save"** to apply changes

### Advanced Options

Access advanced Llama.cpp options:

```yaml
# Example advanced configuration
options:
  rope_freq_base: 10000
  rope_freq_scale: 1.0
  yarn_ext_factor: -1.0
  yarn_attn_factor: 1.0
  yarn_beta_fast: 32.0
  yarn_beta_slow: 1.0
```

## System Information

### Health Dashboard

Monitor overall system health:

- **System Resources**: CPU, memory, disk usage
- **Instance Summary**: Running/stopped instance counts
- **Performance Metrics**: Request rates, response times

### Resource Usage

Track resource consumption:

- Per-instance memory usage
- CPU utilization
- GPU memory (if applicable)
- Network I/O

## User Interface Features

### Theme Support

Switch between light and dark themes:

1. Click the theme toggle button
2. Setting is remembered across sessions

### Responsive Design

The UI adapts to different screen sizes:

- **Desktop**: Full-featured dashboard
- **Tablet**: Condensed layout
- **Mobile**: Stack-based navigation

### Keyboard Shortcuts

- `Ctrl+N`: Create new instance
- `Ctrl+R`: Refresh dashboard
- `Ctrl+L`: Open logs for selected instance
- `Esc`: Close dialogs

## Authentication

### Login

If authentication is enabled:

1. Navigate to the web UI
2. Enter your credentials
3. JWT token is stored for the session
4. Automatic logout on token expiry

### Session Management

- Sessions persist across browser restarts
- Logout clears authentication tokens
- Configurable session timeout

## Troubleshooting

### Common UI Issues

**Page won't load:**
- Check if Llamactl server is running
- Verify the correct URL and port
- Check browser console for errors

**Instance won't start from UI:**
- Verify model path is correct
- Check for port conflicts
- Review instance logs for errors

**Real-time updates not working:**
- Check WebSocket connection
- Verify firewall settings
- Try refreshing the page

### Browser Compatibility

Supported browsers:
- Chrome/Chromium 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Mobile Access

### Responsive Features

On mobile devices:

- Touch-friendly interface
- Swipe gestures for navigation
- Optimized button sizes
- Condensed information display

### Limitations

Some features may be limited on mobile:
- Log viewing (use horizontal scrolling)
- Complex configuration forms
- File browser functionality
