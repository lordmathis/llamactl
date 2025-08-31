# Installation

This guide will walk you through installing Llamactl on your system.

## Prerequisites

You need `llama-server` from [llama.cpp](https://github.com/ggml-org/llama.cpp) installed:

```bash
# Quick install methods:
# Homebrew (macOS)
brew install llama.cpp

# Or build from source - see llama.cpp docs
```

Additional requirements for building from source:
- Go 1.24 or later
- Node.js 22 or later
- Git
- Sufficient disk space for your models

## Installation Methods

### Option 1: Download Binary (Recommended)

Download the latest release from the [GitHub releases page](https://github.com/lordmathis/llamactl/releases):

```bash
# Linux/macOS - Get latest version and download
LATEST_VERSION=$(curl -s https://api.github.com/repos/lordmathis/llamactl/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -L https://github.com/lordmathis/llamactl/releases/download/${LATEST_VERSION}/llamactl-${LATEST_VERSION}-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz | tar -xz
sudo mv llamactl /usr/local/bin/

# Or download manually from:
# https://github.com/lordmathis/llamactl/releases/latest

# Windows - Download from releases page
```

### Option 2: Build from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/lordmathis/llamactl.git
cd llamactl

# Build the web UI
cd webui && npm ci && npm run build && cd ..

# Build the application
go build -o llamactl ./cmd/server
```

## Verification

Verify your installation by checking the version:

```bash
llamactl --version
```

## Next Steps

Now that Llamactl is installed, continue to the [Quick Start](quick-start.md) guide to get your first instance running!
