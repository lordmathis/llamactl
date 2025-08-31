# Installation

This guide will walk you through installing Llamactl on your system.

## Prerequisites

Before installing Llamactl, ensure you have:

- Go 1.19 or later
- Git
- Sufficient disk space for your models

## Installation Methods

### Option 1: Download Binary (Recommended)

Download the latest release from our [GitHub releases page](https://github.com/lordmathis/llamactl/releases):

```bash
# Download for Linux
curl -L https://github.com/lordmathis/llamactl/releases/latest/download/llamactl-linux-amd64 -o llamactl

# Make executable
chmod +x llamactl

# Move to PATH (optional)
sudo mv llamactl /usr/local/bin/
```

### Option 2: Build from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/lordmathis/llamactl.git
cd llamactl

# Build the application
go build -o llamactl cmd/server/main.go
```

For detailed build instructions, see the [Building from Source](../development/building.md) guide.

## Verification

Verify your installation by checking the version:

```bash
llamactl --version
```

## Next Steps

Now that Llamactl is installed, continue to the [Quick Start](quick-start.md) guide to get your first instance running!
