# Contributing to Llamactl

Thank you for considering contributing to Llamactl! This document outlines the development setup and contribution process.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Node.js 22 or later
- `llama-server` executable (from [llama.cpp](https://github.com/ggml-org/llama.cpp))

### Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/lordmathis/llamactl.git
   cd llamactl
   ```

2. **Install dependencies**
   ```bash
   # Go dependencies
   go mod download
   
   # Frontend dependencies
   cd webui && npm ci && cd ..
   ```

3. **Run for development**
   ```bash
   # Start backend server
   go run ./cmd/server
   ```
   Server will be available at `http://localhost:8080`
   
   ```bash
   # In a separate terminal, start frontend dev server
   cd webui && npm run dev
   ```
   Development UI will be available at `http://localhost:5173`

4. **Common development commands**
   ```bash
   # Backend
   go test ./... -v                    # Run tests
   go test -race ./... -v              # Run with race detector
   go fmt ./... && go vet ./...        # Format and vet code
   
   # Frontend (run from webui/ directory)
   npm run test:run                    # Run tests once
   npm run test                        # Run tests in watch mode
   npm run type-check                  # TypeScript type checking
   npm run lint:fix                    # Lint and fix issues
   ```

## Before Submitting a Pull Request

### Required Checks

All the following must pass:

1. **Backend**
   ```bash
   go test ./... -v
   go test -race ./... -v
   go fmt ./... && go vet ./...
   go build -o llamactl ./cmd/server
   ```

2. **Frontend**
   ```bash
   cd webui
   npm run test:run
   npm run type-check
   npm run build
   ```

### API Documentation

If changes affect API endpoints, update Swagger documentation:

```bash
# Install swag if needed
go install github.com/swaggo/swag/cmd/swag@latest

# Update Swagger comments in pkg/server/handlers.go
# Then regenerate docs
swag init -g cmd/server/main.go
```

## Pull Request Guidelines

### Pull Request Titles
Use this format for pull request titles:
- `feat:` for new features
- `fix:` for bug fixes  
- `docs:` for documentation changes
- `test:` for test additions or modifications
- `refactor:` for code refactoring

### Submission Process
1. Create a feature branch from `main`
2. Make changes following the coding standards
3. Run all required checks listed above
4. Update documentation if necessary
5. Submit pull request with:
   - Clear description of changes
   - Reference to any related issues
   - Screenshots for UI changes

## Code Style and Testing

### Testing Strategy
- Backend tests use Go's built-in testing framework
- Frontend tests use Vitest and React Testing Library
- Run tests frequently during development
- Add tests for new features and bug fixes

### Go
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names  
- Add comments for exported functions and types
- Handle errors appropriately

### TypeScript/React
- Use TypeScript strictly (avoid `any` when possible)
- Follow React hooks best practices
- Use meaningful component and variable names
- Prefer functional components over class components

## Documentation Development

This project uses MkDocs for documentation. When working on documentation:

### Setup Documentation Environment

```bash
# Install documentation dependencies
pip install -r docs-requirements.txt
```

### Development Workflow

```bash
# Serve documentation locally for development
mkdocs serve
```
The documentation will be available at http://localhost:8000

```bash
# Build static documentation site
mkdocs build
```
The built site will be in the `site/` directory.

### Documentation Structure

- `docs/` - Documentation content (Markdown files)
- `mkdocs.yml` - MkDocs configuration
- `docs-requirements.txt` - Python dependencies for documentation

### Adding New Documentation

When adding new documentation:

1. Create Markdown files in the appropriate `docs/` subdirectory
2. Update the navigation in `mkdocs.yml`
3. Test locally with `mkdocs serve`
4. Submit a pull request

### Documentation Deployment

Documentation is automatically built and deployed to GitHub Pages when changes are pushed to the main branch.

## Getting Help

- Check existing [issues](https://github.com/lordmathis/llamactl/issues)
- Review the [README.md](README.md) for usage documentation  
- Look at existing code for patterns and conventions

Thank you for contributing to Llamactl!