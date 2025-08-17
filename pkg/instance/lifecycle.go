package instance

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// Start starts the llama server instance and returns an error if it fails.
func (i *Process) Start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Running {
		return fmt.Errorf("instance %s is already running", i.Name)
	}

	// Safety check: ensure options are valid
	if i.options == nil {
		return fmt.Errorf("instance %s has no options set", i.Name)
	}

	// Reset restart counter when manually starting (not during auto-restart)
	// We can detect auto-restart by checking if restartCancel is set
	if i.restartCancel == nil {
		i.restarts = 0
	}

	// Initialize last request time to current time when starting
	i.lastRequestTime.Store(time.Now().Unix())

	// Create log files
	if err := i.logger.Create(); err != nil {
		return fmt.Errorf("failed to create log files: %w", err)
	}

	args := i.options.BuildCommandArgs()

	i.ctx, i.cancel = context.WithCancel(context.Background())
	i.cmd = exec.CommandContext(i.ctx, "llama-server", args...)

	if runtime.GOOS != "windows" {
		setProcAttrs(i.cmd)
	}

	var err error
	i.stdout, err = i.cmd.StdoutPipe()
	if err != nil {
		i.logger.Close()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	i.stderr, err = i.cmd.StderrPipe()
	if err != nil {
		i.stdout.Close()
		i.logger.Close()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := i.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start instance %s: %w", i.Name, err)
	}

	i.Running = true

	// Create channel for monitor completion signaling
	i.monitorDone = make(chan struct{})

	go i.logger.readOutput(i.stdout)
	go i.logger.readOutput(i.stderr)

	go i.monitorProcess()

	return nil
}

// Stop terminates the subprocess
func (i *Process) Stop() error {
	i.mu.Lock()

	if !i.Running {
		// Even if not running, cancel any pending restart
		if i.restartCancel != nil {
			i.restartCancel()
			i.restartCancel = nil
			log.Printf("Cancelled pending restart for instance %s", i.Name)
		}
		i.mu.Unlock()
		return fmt.Errorf("instance %s is not running", i.Name)
	}

	// Cancel any pending restart
	if i.restartCancel != nil {
		i.restartCancel()
		i.restartCancel = nil
	}

	// Set running to false first to signal intentional stop
	i.Running = false

	// Clean up the proxy
	i.proxy = nil

	// Get the monitor done channel before releasing the lock
	monitorDone := i.monitorDone

	i.mu.Unlock()

	// Stop the process with SIGINT
	if i.cmd.Process != nil {
		if err := i.cmd.Process.Signal(syscall.SIGINT); err != nil {
			log.Printf("Failed to send SIGINT to instance %s: %v", i.Name, err)
		}
	}

	select {
	case <-monitorDone:
		// Process exited normally
	case <-time.After(30 * time.Second):
		// Force kill if it doesn't exit within 30 seconds
		if i.cmd.Process != nil {
			killErr := i.cmd.Process.Kill()
			if killErr != nil {
				log.Printf("Failed to force kill instance %s: %v", i.Name, killErr)
			}
			log.Printf("Instance %s did not stop in time, force killed", i.Name)

			// Wait a bit more for the monitor to finish after force kill
			select {
			case <-monitorDone:
				// Monitor completed after force kill
			case <-time.After(2 * time.Second):
				log.Printf("Warning: Monitor goroutine did not complete after force kill for instance %s", i.Name)
			}
		}
	}

	i.logger.Close()

	return nil
}

func (i *Process) monitorProcess() {
	defer func() {
		i.mu.Lock()
		if i.monitorDone != nil {
			close(i.monitorDone)
			i.monitorDone = nil
		}
		i.mu.Unlock()
	}()

	err := i.cmd.Wait()

	i.mu.Lock()

	// Check if the instance was intentionally stopped
	if !i.Running {
		i.mu.Unlock()
		return
	}

	i.Running = false
	i.logger.Close()

	// Cancel any existing restart context since we're handling a new exit
	if i.restartCancel != nil {
		i.restartCancel()
		i.restartCancel = nil
	}

	// Log the exit
	if err != nil {
		log.Printf("Instance %s crashed with error: %v", i.Name, err)
		// Handle restart while holding the lock, then release it
		i.handleRestart()
	} else {
		log.Printf("Instance %s exited cleanly", i.Name)
		i.mu.Unlock()
	}
}

// handleRestart manages the restart process while holding the lock
func (i *Process) handleRestart() {
	// Validate restart conditions and get safe parameters
	shouldRestart, maxRestarts, restartDelay := i.validateRestartConditions()
	if !shouldRestart {
		i.mu.Unlock()
		return
	}

	i.restarts++
	log.Printf("Auto-restarting instance %s (attempt %d/%d) in %v",
		i.Name, i.restarts, maxRestarts, time.Duration(restartDelay)*time.Second)

	// Create a cancellable context for the restart delay
	restartCtx, cancel := context.WithCancel(context.Background())
	i.restartCancel = cancel

	// Release the lock before sleeping
	i.mu.Unlock()

	// Use context-aware sleep so it can be cancelled
	select {
	case <-time.After(time.Duration(restartDelay) * time.Second):
		// Sleep completed normally, continue with restart
	case <-restartCtx.Done():
		// Restart was cancelled
		log.Printf("Restart cancelled for instance %s", i.Name)
		return
	}

	// Restart the instance
	if err := i.Start(); err != nil {
		log.Printf("Failed to restart instance %s: %v", i.Name, err)
	} else {
		log.Printf("Successfully restarted instance %s", i.Name)
		// Clear the cancel function
		i.mu.Lock()
		i.restartCancel = nil
		i.mu.Unlock()
	}
}

// validateRestartConditions checks if the instance should be restarted and returns the parameters
func (i *Process) validateRestartConditions() (shouldRestart bool, maxRestarts int, restartDelay int) {
	if i.options == nil {
		log.Printf("Instance %s not restarting: options are nil", i.Name)
		return false, 0, 0
	}

	if i.options.AutoRestart == nil || !*i.options.AutoRestart {
		log.Printf("Instance %s not restarting: AutoRestart is disabled", i.Name)
		return false, 0, 0
	}

	if i.options.MaxRestarts == nil {
		log.Printf("Instance %s not restarting: MaxRestarts is nil", i.Name)
		return false, 0, 0
	}

	if i.options.RestartDelay == nil {
		log.Printf("Instance %s not restarting: RestartDelay is nil", i.Name)
		return false, 0, 0
	}

	// Values are already validated during unmarshaling/SetOptions
	maxRestarts = *i.options.MaxRestarts
	restartDelay = *i.options.RestartDelay

	if i.restarts >= maxRestarts {
		log.Printf("Instance %s exceeded max restart attempts (%d)", i.Name, maxRestarts)
		return false, 0, 0
	}

	return true, maxRestarts, restartDelay
}
