package instance

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"llamactl/pkg/backends"
	"llamactl/pkg/config"
)

// process manages the OS process lifecycle for a local instance (unexported).
// process owns its complete lifecycle including auto-restart logic.
type process struct {
	instance *Instance // Back-reference for SetStatus, GetOptions

	mu            sync.RWMutex
	cmd           *exec.Cmd
	ctx           context.Context
	cancel        context.CancelFunc
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	restarts      int               // process owns restart counter
	restartCancel context.CancelFunc
	monitorDone   chan struct{}
}

// newProcess creates a new process component for the given instance
func newProcess(instance *Instance) *process {
	return &process{
		instance: instance,
	}
}

// Start starts the OS process and returns an error if it fails.
func (p *process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.instance.IsRunning() {
		return fmt.Errorf("instance %s is already running", p.instance.Name)
	}

	// Safety check: ensure options are valid
	if p.instance.options == nil {
		return fmt.Errorf("instance %s has no options set", p.instance.Name)
	}

	// Reset restart counter when manually starting (not during auto-restart)
	// We can detect auto-restart by checking if restartCancel is set
	if p.restartCancel == nil {
		p.restarts = 0
	}

	// Initialize last request time to current time when starting
	if p.instance.proxy != nil {
		p.instance.proxy.UpdateLastRequestTime()
	}

	// Create context before building command (needed for CommandContext)
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Create log files
	if err := p.instance.logger.Create(); err != nil {
		return fmt.Errorf("failed to create log files: %w", err)
	}

	// Build command using backend-specific methods
	cmd, cmdErr := p.buildCommand()
	if cmdErr != nil {
		return fmt.Errorf("failed to build command: %w", cmdErr)
	}
	p.cmd = cmd

	if runtime.GOOS != "windows" {
		setProcAttrs(p.cmd)
	}

	var err error
	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		p.instance.logger.Close()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		p.stdout.Close()
		p.instance.logger.Close()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start instance %s: %w", p.instance.Name, err)
	}

	p.instance.SetStatus(Running)

	// Create channel for monitor completion signaling
	p.monitorDone = make(chan struct{})

	go p.instance.logger.readOutput(p.stdout)
	go p.instance.logger.readOutput(p.stderr)

	go p.monitorProcess()

	return nil
}

// Stop terminates the subprocess without restarting
func (p *process) Stop() error {
	p.mu.Lock()

	if !p.instance.IsRunning() {
		// Even if not running, cancel any pending restart
		if p.restartCancel != nil {
			p.restartCancel()
			p.restartCancel = nil
			log.Printf("Cancelled pending restart for instance %s", p.instance.Name)
		}
		p.mu.Unlock()
		return fmt.Errorf("instance %s is not running", p.instance.Name)
	}

	// Cancel any pending restart
	if p.restartCancel != nil {
		p.restartCancel()
		p.restartCancel = nil
	}

	// Set status to stopped first to signal intentional stop
	p.instance.SetStatus(Stopped)

	// Get the monitor done channel before releasing the lock
	monitorDone := p.monitorDone

	p.mu.Unlock()

	// Stop the process with SIGINT if cmd exists
	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Signal(syscall.SIGINT); err != nil {
			log.Printf("Failed to send SIGINT to instance %s: %v", p.instance.Name, err)
		}
	}

	// If no process exists, we can return immediately
	if p.cmd == nil || monitorDone == nil {
		p.instance.logger.Close()
		return nil
	}

	select {
	case <-monitorDone:
		// Process exited normally
	case <-time.After(30 * time.Second):
		// Force kill if it doesn't exit within 30 seconds
		if p.cmd != nil && p.cmd.Process != nil {
			killErr := p.cmd.Process.Kill()
			if killErr != nil {
				log.Printf("Failed to force kill instance %s: %v", p.instance.Name, killErr)
			}
			log.Printf("Instance %s did not stop in time, force killed", p.instance.Name)

			// Wait a bit more for the monitor to finish after force kill
			select {
			case <-monitorDone:
				// Monitor completed after force kill
			case <-time.After(2 * time.Second):
				log.Printf("Warning: Monitor goroutine did not complete after force kill for instance %s", p.instance.Name)
			}
		}
	}

	p.instance.logger.Close()

	return nil
}

// Restart manually restarts the process (resets restart counter)
func (p *process) Restart() error {
	// Stop the process first
	if err := p.Stop(); err != nil {
		// If it's not running, that's ok - we'll just start it
		if err.Error() != fmt.Sprintf("instance %s is not running", p.instance.Name) {
			return fmt.Errorf("failed to stop instance during restart: %w", err)
		}
	}

	// Reset restart counter for manual restart
	p.mu.Lock()
	p.restarts = 0
	p.mu.Unlock()

	// Start the process
	return p.Start()
}

// WaitForHealthy waits for the process to become healthy
func (p *process) WaitForHealthy(timeout int) error {
	if !p.instance.IsRunning() {
		return fmt.Errorf("instance %s is not running", p.instance.Name)
	}

	if timeout <= 0 {
		timeout = 30 // Default to 30 seconds if no timeout is specified
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Get host/port from instance
	host, port := p.instance.getBackendHostPort()
	healthURL := fmt.Sprintf("http://%s:%d/health", host, port)

	// Create a dedicated HTTP client for health checks
	client := &http.Client{
		Timeout: 5 * time.Second, // 5 second timeout per request
	}

	// Helper function to check health directly
	checkHealth := func() bool {
		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			return false
		}

		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		return resp.StatusCode == http.StatusOK
	}

	// Try immediate check first
	if checkHealth() {
		return nil // Instance is healthy
	}

	// If immediate check failed, start polling
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for instance %s to become healthy after %d seconds", p.instance.Name, timeout)
		case <-ticker.C:
			if checkHealth() {
				return nil // Instance is healthy
			}
			// Continue polling
		}
	}
}

// monitorProcess monitors the OS process and handles crashes/exits
func (p *process) monitorProcess() {
	defer func() {
		p.mu.Lock()
		if p.monitorDone != nil {
			close(p.monitorDone)
			p.monitorDone = nil
		}
		p.mu.Unlock()
	}()

	err := p.cmd.Wait()

	p.mu.Lock()

	// Check if the instance was intentionally stopped
	if !p.instance.IsRunning() {
		p.mu.Unlock()
		return
	}

	p.instance.SetStatus(Stopped)
	p.instance.logger.Close()

	// Cancel any existing restart context since we're handling a new exit
	if p.restartCancel != nil {
		p.restartCancel()
		p.restartCancel = nil
	}

	// Log the exit
	if err != nil {
		log.Printf("Instance %s crashed with error: %v", p.instance.Name, err)
		// Handle auto-restart logic
		p.handleAutoRestart(err)
	} else {
		log.Printf("Instance %s exited cleanly", p.instance.Name)
		p.mu.Unlock()
	}
}

// shouldAutoRestart checks if the process should auto-restart
func (p *process) shouldAutoRestart() bool {
	opts := p.instance.GetOptions()
	if opts == nil {
		log.Printf("Instance %s not restarting: options are nil", p.instance.Name)
		return false
	}

	if opts.AutoRestart == nil || !*opts.AutoRestart {
		log.Printf("Instance %s not restarting: AutoRestart is disabled", p.instance.Name)
		return false
	}

	if opts.MaxRestarts == nil {
		log.Printf("Instance %s not restarting: MaxRestarts is nil", p.instance.Name)
		return false
	}

	maxRestarts := *opts.MaxRestarts
	if p.restarts >= maxRestarts {
		log.Printf("Instance %s exceeded max restart attempts (%d)", p.instance.Name, maxRestarts)
		return false
	}

	return true
}

// handleAutoRestart manages the auto-restart process
func (p *process) handleAutoRestart(err error) {
	// Check if should restart
	if !p.shouldAutoRestart() {
		p.instance.SetStatus(Failed)
		p.mu.Unlock()
		return
	}

	// Get restart parameters
	opts := p.instance.GetOptions()
	if opts.RestartDelay == nil {
		log.Printf("Instance %s not restarting: RestartDelay is nil", p.instance.Name)
		p.instance.SetStatus(Failed)
		p.mu.Unlock()
		return
	}

	restartDelay := *opts.RestartDelay
	maxRestarts := *opts.MaxRestarts

	p.restarts++
	log.Printf("Auto-restarting instance %s (attempt %d/%d) in %v",
		p.instance.Name, p.restarts, maxRestarts, time.Duration(restartDelay)*time.Second)

	// Create a cancellable context for the restart delay
	restartCtx, cancel := context.WithCancel(context.Background())
	p.restartCancel = cancel

	// Release the lock before sleeping
	p.mu.Unlock()

	// Use context-aware sleep so it can be cancelled
	select {
	case <-time.After(time.Duration(restartDelay) * time.Second):
		// Sleep completed normally, continue with restart
	case <-restartCtx.Done():
		// Restart was cancelled
		log.Printf("Restart cancelled for instance %s", p.instance.Name)
		return
	}

	// Restart the instance
	if err := p.Start(); err != nil {
		log.Printf("Failed to restart instance %s: %v", p.instance.Name, err)
	} else {
		log.Printf("Successfully restarted instance %s", p.instance.Name)
		// Clear the cancel function
		p.mu.Lock()
		p.restartCancel = nil
		p.mu.Unlock()
	}
}

// buildCommand builds the command to execute using backend-specific logic
func (p *process) buildCommand() (*exec.Cmd, error) {
	// Get options
	opts := p.instance.GetOptions()
	if opts == nil {
		return nil, fmt.Errorf("instance options are nil")
	}

	// Get backend configuration
	backendConfig, err := p.getBackendConfig()
	if err != nil {
		return nil, err
	}

	// Build the environment variables
	env := opts.BuildEnvironment(backendConfig)

	// Get the command to execute
	command := opts.GetCommand(backendConfig)

	// Build command arguments
	args := opts.BuildCommandArgs(backendConfig)

	// Create the exec.Cmd
	cmd := exec.CommandContext(p.ctx, command, args...)

	// Start with host environment variables
	cmd.Env = os.Environ()

	// Add/override with backend-specific environment variables
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return cmd, nil
}

// getBackendConfig resolves the backend configuration for the current instance
func (p *process) getBackendConfig() (*config.BackendSettings, error) {
	opts := p.instance.GetOptions()
	if opts == nil {
		return nil, fmt.Errorf("instance options are nil")
	}

	var backendTypeStr string

	switch opts.BackendType {
	case backends.BackendTypeLlamaCpp:
		backendTypeStr = "llama-cpp"
	case backends.BackendTypeMlxLm:
		backendTypeStr = "mlx"
	case backends.BackendTypeVllm:
		backendTypeStr = "vllm"
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", opts.BackendType)
	}

	settings := p.instance.globalBackendSettings.GetBackendSettings(backendTypeStr)
	return &settings, nil
}
