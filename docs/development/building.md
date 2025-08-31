# Building from Source

This guide covers building LlamaCtl from source code for development and production deployment.

## Prerequisites

### Required Tools

- **Go 1.24+**: Download from [golang.org](https://golang.org/dl/)
- **Node.js 22+**: Download from [nodejs.org](https://nodejs.org/)
- **Git**: For cloning the repository
- **Make**: For build automation (optional)

### System Requirements

- **Memory**: 4GB+ RAM for building
- **Disk**: 2GB+ free space
- **OS**: Linux, macOS, or Windows

## Quick Build

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/lordmathis/llamactl.git
cd llamactl

# Build the application
go build -o llamactl cmd/server/main.go
```

### Run

```bash
./llamactl
```

## Development Build

### Setup Development Environment

```bash
# Clone repository
git clone https://github.com/lordmathis/llamactl.git
cd llamactl

# Install Go dependencies
go mod download

# Install frontend dependencies
cd webui
npm ci
cd ..
```

### Build Components

```bash
# Build backend only
go build -o llamactl cmd/server/main.go

# Build frontend only
cd webui
npm run build
cd ..

# Build everything
make build
```

### Development Server

```bash
# Run backend in development mode
go run cmd/server/main.go --dev

# Run frontend dev server (separate terminal)
cd webui
npm run dev
```

## Production Build

### Optimized Build

```bash
# Build with optimizations
go build -ldflags="-s -w" -o llamactl cmd/server/main.go

# Or use the Makefile
make build-prod
```

### Build Flags

Common build flags for production:

```bash
go build \
  -ldflags="-s -w -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -trimpath \
  -o llamactl \
  cmd/server/main.go
```

**Flag explanations:**
- `-s`: Strip symbol table
- `-w`: Strip debug information
- `-X`: Set variable values at build time
- `-trimpath`: Remove absolute paths from binary

## Cross-Platform Building

### Build for Multiple Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o llamactl-linux-amd64 cmd/server/main.go

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o llamactl-linux-arm64 cmd/server/main.go

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o llamactl-darwin-amd64 cmd/server/main.go

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o llamactl-darwin-arm64 cmd/server/main.go

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o llamactl-windows-amd64.exe cmd/server/main.go
```

### Automated Cross-Building

Use the provided Makefile:

```bash
# Build all platforms
make build-all

# Build specific platform
make build-linux
make build-darwin
make build-windows
```

## Build with Docker

### Development Container

```dockerfile
# Dockerfile.dev
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o llamactl cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/llamactl .

EXPOSE 8080
CMD ["./llamactl"]
```

```bash
# Build development image
docker build -f Dockerfile.dev -t llamactl:dev .

# Run container
docker run -p 8080:8080 llamactl:dev
```

### Production Container

```dockerfile
# Dockerfile
FROM node:22-alpine AS frontend-builder

WORKDIR /app/webui
COPY webui/package*.json ./
RUN npm ci

COPY webui/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/webui/dist ./webui/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o llamactl \
    cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -s /bin/sh llamactl

WORKDIR /home/llamactl
COPY --from=backend-builder /app/llamactl .
RUN chown llamactl:llamactl llamactl

USER llamactl
EXPOSE 8080

CMD ["./llamactl"]
```

## Advanced Build Options

### Static Linking

For deployments without external dependencies:

```bash
CGO_ENABLED=0 go build \
  -ldflags="-s -w -extldflags '-static'" \
  -o llamactl-static \
  cmd/server/main.go
```

### Debug Build

Build with debug information:

```bash
go build -gcflags="all=-N -l" -o llamactl-debug cmd/server/main.go
```

### Race Detection Build

Build with race detection (development only):

```bash
go build -race -o llamactl-race cmd/server/main.go
```

## Build Automation

### Makefile

```makefile
# Makefile
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

.PHONY: build clean test install

build:
	@echo "Building LlamaCtl..."
	@cd webui && npm run build
	@go build -ldflags="$(LDFLAGS)" -o llamactl cmd/server/main.go

build-prod:
	@echo "Building production binary..."
	@cd webui && npm run build
	@CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o llamactl cmd/server/main.go

build-all: build-linux build-darwin build-windows

build-linux:
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/llamactl-linux-amd64 cmd/server/main.go
	@GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/llamactl-linux-arm64 cmd/server/main.go

build-darwin:
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/llamactl-darwin-amd64 cmd/server/main.go
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/llamactl-darwin-arm64 cmd/server/main.go

build-windows:
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/llamactl-windows-amd64.exe cmd/server/main.go

test:
	@go test ./...

clean:
	@rm -f llamactl llamactl-*
	@rm -rf dist/

install: build
	@cp llamactl $(GOPATH)/bin/llamactl
```

### GitHub Actions

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '22'
    
    - name: Install dependencies
      run: |
        go mod download
        cd webui && npm ci
    
    - name: Run tests
      run: |
        go test ./...
        cd webui && npm test
    
    - name: Build
      run: make build

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '22'
    
    - name: Build all platforms
      run: make build-all
    
    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: binaries
        path: dist/
```

## Build Troubleshooting

### Common Issues

**Go version mismatch:**
```bash
# Check Go version
go version

# Update Go
# Download from https://golang.org/dl/
```

**Node.js issues:**
```bash
# Clear npm cache
npm cache clean --force

# Remove node_modules and reinstall
rm -rf webui/node_modules
cd webui && npm ci
```

**Build failures:**
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Performance Issues

**Slow builds:**
```bash
# Use build cache
export GOCACHE=$(go env GOCACHE)

# Parallel builds
export GOMAXPROCS=$(nproc)
```

**Large binary size:**
```bash
# Use UPX compression
upx --best llamactl

# Analyze binary size
go tool nm -size llamactl | head -20
```

## Deployment

### System Service

Create a systemd service:

```ini
# /etc/systemd/system/llamactl.service
[Unit]
Description=LlamaCtl Server
After=network.target

[Service]
Type=simple
User=llamactl
Group=llamactl
ExecStart=/usr/local/bin/llamactl
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl enable llamactl
sudo systemctl start llamactl
```

### Configuration

```bash
# Create configuration directory
sudo mkdir -p /etc/llamactl

# Copy configuration
sudo cp config.yaml /etc/llamactl/

# Set permissions
sudo chown -R llamactl:llamactl /etc/llamactl
```

## Next Steps

- Configure [Installation](../getting-started/installation.md)
- Set up [Configuration](../getting-started/configuration.md)
- Learn about [Contributing](contributing.md)
