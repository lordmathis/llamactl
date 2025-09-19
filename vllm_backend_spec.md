# vLLM Backend Implementation Specification

## Overview
This specification outlines the implementation of vLLM backend support for llamactl, following the existing patterns established by the llama.cpp and MLX backends.

## 1. Backend Configuration

### Basic Details
- **Backend Type**: `vllm`
- **Executable**: `vllm` (configured via `VllmExecutable`)
- **Subcommand**: `serve` (automatically prepended to arguments)
- **Default Host/Port**: Auto-assigned by llamactl
- **Health Check**: Uses `/health` endpoint (returns HTTP 200 with no content)
- **API Compatibility**: OpenAI-compatible endpoints

### Example Command
```bash
vllm serve --enable-log-outputs --tensor-parallel-size 2 --gpu-memory-utilization 0.5 --model ISTA-DASLab/gemma-3-27b-it-GPTQ-4b-128g
```

## 2. File Structure
Following the existing backend pattern:
```
pkg/backends/vllm/
├── vllm.go          # VllmServerOptions struct and methods
├── vllm_test.go     # Unit tests for VllmServerOptions
├── parser.go        # Command parsing logic
└── parser_test.go   # Parser tests
```

## 3. Core Implementation Files

### 3.1 `pkg/backends/vllm/vllm.go`

#### VllmServerOptions Struct
```go
type VllmServerOptions struct {
    // Basic connection options (auto-assigned by llamactl)
    Host string `json:"host,omitempty"`
    Port int    `json:"port,omitempty"`
    
    // Core model options
    Model string `json:"model,omitempty"`
    
    // Common serving options
    EnableLogOutputs       bool    `json:"enable_log_outputs,omitempty"`
    TensorParallelSize    int     `json:"tensor_parallel_size,omitempty"`
    GPUMemoryUtilization  float64 `json:"gpu_memory_utilization,omitempty"`
    
    // Additional parameters to be added based on vLLM CLI documentation
    // Following the same comprehensive approach as llamacpp.LlamaServerOptions
}
```

#### Required Methods
- `UnmarshalJSON()` - Custom unmarshaling with alternative field name support (dash-to-underscore conversion)
- `BuildCommandArgs()` - Convert struct to command line arguments (excluding "serve" subcommand)
- `NewVllmServerOptions()` - Constructor with vLLM defaults

#### Field Name Mapping
Support both CLI argument names (with dashes) and programmatic names (with underscores), similar to the llama.cpp implementation:
```go
fieldMappings := map[string]string{
    "enable-log-outputs":       "enable_log_outputs",
    "tensor-parallel-size":     "tensor_parallel_size", 
    "gpu-memory-utilization":   "gpu_memory_utilization",
    // ... other mappings
}
```

### 3.2 `pkg/backends/vllm/parser.go`

#### ParseVllmCommand Function
Following the same pattern as `llamacpp/parser.go` and `mlx/parser.go`:

```go
func ParseVllmCommand(command string) (*VllmServerOptions, error)
```

**Supported Input Formats:**
1. `vllm serve --model MODEL_NAME --other-args`
2. `/path/to/vllm serve --model MODEL_NAME`  
3. `serve --model MODEL_NAME --other-args`
4. `--model MODEL_NAME --other-args` (args only)
5. Multiline commands with backslashes

**Implementation Details:**
- Handle "serve" subcommand detection and removal
- Support quoted strings and escaped characters
- Validate command structure
- Convert parsed arguments to `VllmServerOptions`

## 4. Backend Integration

### 4.1 Backend Type Definition
**File**: `pkg/backends/backend.go`
```go
const (
    BackendTypeLlamaCpp BackendType = "llama_cpp"
    BackendTypeMlxLm    BackendType = "mlx_lm" 
    BackendTypeVllm     BackendType = "vllm"     // ADD THIS
)
```

### 4.2 Configuration Integration
**File**: `pkg/config/config.go`

#### BackendConfig Update
```go
type BackendConfig struct {
    LlamaExecutable string `yaml:"llama_executable"`
    MLXLMExecutable string `yaml:"mlx_lm_executable"`
    VllmExecutable  string `yaml:"vllm_executable"`  // ADD THIS
}
```

#### Default Configuration
- **Default Value**: `"vllm"`
- **Environment Variable**: `LLAMACTL_VLLM_EXECUTABLE`

#### Environment Variable Loading
Add to `loadEnvVars()` function:
```go
if vllmExec := os.Getenv("LLAMACTL_VLLM_EXECUTABLE"); vllmExec != "" {
    cfg.Backends.VllmExecutable = vllmExec
}
```

### 4.3 Instance Options Integration
**File**: `pkg/instance/options.go`

#### CreateInstanceOptions Update
```go
type CreateInstanceOptions struct {
    // existing fields...
    VllmServerOptions *vllm.VllmServerOptions `json:"-"`
}
```

#### JSON Marshaling/Unmarshaling
Update `UnmarshalJSON()` and `MarshalJSON()` methods to handle vLLM backend similar to existing backends.

#### BuildCommandArgs Implementation
```go
case backends.BackendTypeVllm:
    if c.VllmServerOptions != nil {
        // Prepend "serve" as first argument
        args := []string{"serve"}
        args = append(args, c.VllmServerOptions.BuildCommandArgs()...)
        return args
    }
```

**Key Point**: The "serve" subcommand is handled at the instance options level, keeping the `VllmServerOptions.BuildCommandArgs()` method focused only on vLLM-specific parameters.

## 5. Health Check Integration

### 5.1 Standard Health Check for vLLM
**File**: `pkg/instance/lifecycle.go`

vLLM provides a standard `/health` endpoint that returns HTTP 200 with no content, so no modifications are needed to the existing health check logic. The current `WaitForHealthy()` method will work as-is:

```go
healthURL := fmt.Sprintf("http://%s:%d/health", host, port)
```

### 5.2 Startup Time Considerations
- vLLM typically has longer startup times compared to llama.cpp
- The existing configurable timeout system should handle this adequately
- Users may need to adjust `on_demand_start_timeout` for larger models

## 6. Lifecycle Integration

### 6.1 Executable Selection
**File**: `pkg/instance/lifecycle.go`

Update the `Start()` method to handle vLLM executable:

```go
switch i.options.BackendType {
case backends.BackendTypeLlamaCpp:
    executable = i.globalBackendSettings.LlamaExecutable
case backends.BackendTypeMlxLm:
    executable = i.globalBackendSettings.MLXLMExecutable
case backends.BackendTypeVllm:                              // ADD THIS
    executable = i.globalBackendSettings.VllmExecutable
default:
    return fmt.Errorf("unsupported backend type: %s", i.options.BackendType)
}

args := i.options.BuildCommandArgs()
i.cmd = exec.CommandContext(i.ctx, executable, args...)
```

### 6.2 Command Execution
The final executed command will be:
```bash
vllm serve --model MODEL_NAME --other-vllm-args
```

Where:
- `vllm` comes from `VllmExecutable` configuration
- `serve` is prepended by `BuildCommandArgs()`
- Remaining args come from `VllmServerOptions.BuildCommandArgs()`

## 7. Server Handler Integration

### 7.1 New Handler Method
**File**: `pkg/server/handlers.go`

```go
// ParseVllmCommand godoc
// @Summary Parse vllm serve command
// @Description Parses a vLLM serve command string into instance options
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.CreateInstanceOptions "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /backends/vllm/parse-command [post]
func (h *Handler) ParseVllmCommand() http.HandlerFunc {
    // Implementation similar to ParseMlxCommand()
    // Uses vllm.ParseVllmCommand() internally
}
```

### 7.2 Router Integration
**File**: `pkg/server/routes.go`

Add vLLM route:
```go
r.Route("/backends", func(r chi.Router) {
    r.Route("/llama-cpp", func(r chi.Router) {
        r.Post("/parse-command", handler.ParseLlamaCommand())
    })
    r.Route("/mlx", func(r chi.Router) {
        r.Post("/parse-command", handler.ParseMlxCommand())
    })
    r.Route("/vllm", func(r chi.Router) {      // ADD THIS
        r.Post("/parse-command", handler.ParseVllmCommand())
    })
})
```

## 8. Validation Integration

### 8.1 Instance Options Validation
**File**: `pkg/validation/validation.go`

Add vLLM validation case:
```go
func ValidateInstanceOptions(options *instance.CreateInstanceOptions) error {
    // existing validation...
    
    switch options.BackendType {
    case backends.BackendTypeLlamaCpp:
        return validateLlamaCppOptions(options)
    case backends.BackendTypeMlxLm:
        return validateMlxOptions(options)
    case backends.BackendTypeVllm:          // ADD THIS
        return validateVllmOptions(options)
    default:
        return ValidationError(fmt.Errorf("unsupported backend type: %s", options.BackendType))
    }
}

func validateVllmOptions(options *instance.CreateInstanceOptions) error {
    if options.VllmServerOptions == nil {
        return ValidationError(fmt.Errorf("vLLM server options cannot be nil for vLLM backend"))
    }
    
    // Basic validation following the same pattern as other backends
    if err := validateStructStrings(options.VllmServerOptions, ""); err != nil {
        return err
    }
    
    // Port validation
    if options.VllmServerOptions.Port < 0 || options.VllmServerOptions.Port > 65535 {
        return ValidationError(fmt.Errorf("invalid port range: %d", options.VllmServerOptions.Port))
    }
    
    return nil
}
```

## 9. Testing Strategy

### 9.1 Unit Tests
- **`vllm_test.go`**: Test `VllmServerOptions` marshaling/unmarshaling, BuildCommandArgs()
- **`parser_test.go`**: Test command parsing for various formats
- **Integration tests**: Mock vLLM commands and validate parsing

### 9.2 Test Cases
```go
func TestBuildCommandArgs_VllmBasic(t *testing.T) {
    options := VllmServerOptions{
        Model:              "microsoft/DialoGPT-medium",
        Port:               8080,
        Host:               "localhost", 
        EnableLogOutputs:   true,
        TensorParallelSize: 2,
    }
    
    args := options.BuildCommandArgs()
    // Validate expected arguments (excluding "serve")
}

func TestParseVllmCommand_FullCommand(t *testing.T) {
    command := "vllm serve --model ISTA-DASLab/gemma-3-27b-it-GPTQ-4b-128g --tensor-parallel-size 2"
    result, err := ParseVllmCommand(command)
    // Validate parsing results
}
```

## 10. Example Usage

### 10.1 Parse Existing vLLM Command
```bash
curl -X POST http://localhost:8080/api/v1/backends/vllm/parse-command \
  -H "Authorization: Bearer your-management-key" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "vllm serve --model ISTA-DASLab/gemma-3-27b-it-GPTQ-4b-128g --tensor-parallel-size 2 --gpu-memory-utilization 0.5"
  }'
```

### 10.2 Create vLLM Instance
```bash
curl -X POST http://localhost:8080/api/v1/instances/my-vllm-model \
  -H "Authorization: Bearer your-management-key" \
  -H "Content-Type: application/json" \
  -d '{
    "backend_type": "vllm",
    "backend_options": {
      "model": "ISTA-DASLab/gemma-3-27b-it-GPTQ-4b-128g",
      "tensor_parallel_size": 2,
      "gpu_memory_utilization": 0.5,
      "enable_log_outputs": true
    }
  }'
```

### 10.3 Use via OpenAI-Compatible API
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-inference-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-vllm-model",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## 11. Implementation Checklist

### Phase 1: Core Backend
- [ ] Create `pkg/backends/vllm/vllm.go`
- [ ] Implement `VllmServerOptions` struct with basic fields
- [ ] Implement `BuildCommandArgs()`, `UnmarshalJSON()`, `MarshalJSON()`
- [ ] Add comprehensive field mappings for CLI args
- [ ] Create unit tests for `VllmServerOptions`

### Phase 2: Command Parsing
- [ ] Create `pkg/backends/vllm/parser.go`  
- [ ] Implement `ParseVllmCommand()` function
- [ ] Handle various command input formats
- [ ] Create comprehensive parser tests
- [ ] Test edge cases and error conditions

### Phase 3: Integration
- [ ] Add `BackendTypeVllm` to `pkg/backends/backend.go`
- [ ] Update `BackendConfig` in `pkg/config/config.go`
- [ ] Add environment variable support
- [ ] Update `CreateInstanceOptions` in `pkg/instance/options.go`
- [ ] Implement `BuildCommandArgs()` with "serve" prepending

### Phase 4: Lifecycle & Health Checks
- [ ] Update executable selection in `pkg/instance/lifecycle.go`
- [ ] Test instance startup and health checking (uses existing `/health` endpoint)
- [ ] Validate command execution flow

### Phase 5: API Integration
- [ ] Add `ParseVllmCommand()` handler in `pkg/server/handlers.go`
- [ ] Add vLLM route in `pkg/server/routes.go`
- [ ] Update validation in `pkg/validation/validation.go`
- [ ] Test API endpoints

### Phase 6: Testing & Documentation
- [ ] Create comprehensive integration tests
- [ ] Test with actual vLLM installation (if available)
- [ ] Update documentation
- [ ] Test OpenAI-compatible proxy functionality

## 12. Configuration Examples

### 12.1 YAML Configuration
```yaml
backends:
  llama_executable: "llama-server"
  mlx_lm_executable: "mlx_lm.server"
  vllm_executable: "vllm"

instances:
  # ... other instance settings
```

### 12.2 Environment Variables
```bash
export LLAMACTL_VLLM_EXECUTABLE="vllm"
# OR for custom installation
export LLAMACTL_VLLM_EXECUTABLE="python -m vllm" 
# OR for containerized deployment
export LLAMACTL_VLLM_EXECUTABLE="docker run --rm --gpus all vllm/vllm-openai"
```

## 13. Notes and Considerations

### 13.1 Startup Time
- vLLM instances may take significantly longer to start than llama.cpp
- Consider documenting recommended timeout values
- The configurable `on_demand_start_timeout` should accommodate this

### 13.2 Resource Usage  
- vLLM typically requires substantial GPU memory
- No special handling needed in llamactl (follows existing pattern)
- Resource management is left to the user/administrator

### 13.3 Model Compatibility
- Primarily designed for HuggingFace models
- Supports various quantization formats (GPTQ, AWQ, etc.)
- Model path validation can be basic (similar to other backends)

### 13.4 Future Enhancements
- Consider adding vLLM-specific parameter validation
- Could add model download/caching features  
- May want to add vLLM version detection capabilities

This specification provides a comprehensive roadmap for implementing vLLM backend support while maintaining consistency with the existing llamactl architecture.