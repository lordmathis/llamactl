# Contributing

Thank you for your interest in contributing to Llamactl! This guide will help you get started with development and contribution.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Node.js 22 or later
- `llama-server` executable (from [llama.cpp](https://github.com/ggml-org/llama.cpp))
- Git

### Getting Started

1. **Fork and Clone**
   ```bash
   # Fork the repository on GitHub, then clone your fork
   git clone https://github.com/yourusername/llamactl.git
   cd llamactl
   
   # Add upstream remote
   git remote add upstream https://github.com/lordmathis/llamactl.git
   ```

2. **Install Dependencies**
   ```bash
   # Go dependencies
   go mod download
   
   # Frontend dependencies
   cd webui && npm ci && cd ..
   ```

3. **Run Development Environment**
   ```bash
   # Start backend server
   go run ./cmd/server
   ```
   
   In a separate terminal:
   ```bash
   # Start frontend dev server
   cd webui && npm run dev
   ```

## Development Workflow

### Setting Up Your Environment

1. **Configuration**
   Create a development configuration file:
   ```yaml
   # dev-config.yaml
   server:
     host: "localhost"
     port: 8080
   logging:
     level: "debug"
   ```

2. **Test Data**
   Set up test models and instances for development.

### Making Changes

1. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Development Commands**
   ```bash
   # Backend
   go test ./... -v                    # Run tests
   go test -race ./... -v              # Run with race detector
   go fmt ./... && go vet ./...        # Format and vet code
   go build ./cmd/server               # Build binary
   
   # Frontend (from webui/ directory)
   npm run test                        # Run tests
   npm run lint                        # Lint code
   npm run type-check                  # TypeScript check
   npm run build                       # Build for production
   ```

3. **Code Quality**
   ```bash
   # Run all checks before committing
   make lint
   make test
   make build
   ```

## Project Structure

### Backend (Go)

```
cmd/
├── server/              # Main application entry point
pkg/
├── backends/           # Model backend implementations
├── config/            # Configuration management
├── instance/          # Instance lifecycle management
├── manager/           # Instance manager
├── server/            # HTTP server and routes
├── testutil/          # Test utilities
└── validation/        # Input validation
```

### Frontend (React/TypeScript)

```
webui/src/
├── components/        # React components
├── contexts/         # React contexts
├── hooks/           # Custom hooks
├── lib/             # Utility libraries
├── schemas/         # Zod schemas
└── types/           # TypeScript types
```

## Coding Standards

### Go Code

- Follow standard Go formatting (`gofmt`)
- Use `go vet` and address all warnings
- Write comprehensive tests for new functionality
- Include documentation comments for exported functions
- Use meaningful variable and function names

Example:
```go
// CreateInstance creates a new model instance with the given configuration.
// It validates the configuration and ensures the instance name is unique.
func (m *Manager) CreateInstance(ctx context.Context, config InstanceConfig) (*Instance, error) {
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    // Implementation...
}
```

### TypeScript/React Code

- Use TypeScript strict mode
- Follow React best practices
- Use functional components with hooks
- Implement proper error boundaries
- Write unit tests for components

Example:
```typescript
interface InstanceCardProps {
  instance: Instance;
  onStart: (name: string) => Promise<void>;
  onStop: (name: string) => Promise<void>;
}

export const InstanceCard: React.FC<InstanceCardProps> = ({
  instance,
  onStart,
  onStop,
}) => {
  // Implementation...
};
```

## Testing

### Backend Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/manager -v

# Run with race detection
go test -race ./...
```

### Frontend Tests

```bash
cd webui

# Run unit tests
npm run test

# Run tests with coverage
npm run test:coverage

# Run E2E tests
npm run test:e2e
```

### Integration Tests

```bash
# Run integration tests (requires llama-server)
go test ./... -tags=integration
```

## Pull Request Process

### Before Submitting

1. **Update your branch**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all tests**
   ```bash
   make test-all
   ```

3. **Update documentation** if needed

4. **Write clear commit messages**
   ```
   feat: add instance health monitoring
   
   - Implement health check endpoint
   - Add periodic health monitoring
   - Update API documentation
   
   Fixes #123
   ```

### Submitting a PR

1. **Push your branch**
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create Pull Request**
   - Use the PR template
   - Provide clear description
   - Link related issues
   - Add screenshots for UI changes

3. **PR Review Process**
   - Automated checks must pass
   - Code review by maintainers
   - Address feedback promptly
   - Keep PR scope focused

## Issue Guidelines

### Reporting Bugs

Use the bug report template and include:

- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages
- Minimal reproduction case

### Feature Requests

Use the feature request template and include:

- Clear description of the problem
- Proposed solution
- Alternative solutions considered
- Implementation complexity estimate

### Security Issues

For security vulnerabilities:
- Do NOT create public issues
- Email security@llamactl.dev
- Provide detailed description
- Allow time for fix before disclosure

## Development Best Practices

### API Design

- Follow REST principles
- Use consistent naming conventions
- Provide comprehensive error messages
- Include proper HTTP status codes
- Document all endpoints

### Error Handling

```go
// Wrap errors with context
if err := instance.Start(); err != nil {
    return fmt.Errorf("failed to start instance %s: %w", instance.Name, err)
}

// Use structured logging
log.WithFields(log.Fields{
    "instance": instance.Name,
    "error": err,
}).Error("Failed to start instance")
```

### Configuration

- Use environment variables for deployment
- Provide sensible defaults
- Validate configuration on startup
- Support configuration file reloading

### Performance

- Profile code for bottlenecks
- Use efficient data structures
- Implement proper caching
- Monitor resource usage

## Release Process

### Version Management

- Use semantic versioning (SemVer)
- Tag releases properly
- Maintain CHANGELOG.md
- Create release notes

### Building Releases

```bash
# Build all platforms
make build-all

# Create release package
make package
```

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Code Review**: PR comments and feedback

### Development Questions

When asking for help:

1. Check existing documentation
2. Search previous issues
3. Provide minimal reproduction case
4. Include relevant environment details

## Recognition

Contributors are recognized in:

- CONTRIBUTORS.md file
- Release notes
- Documentation credits
- Annual contributor highlights

Thank you for contributing to Llamactl!
