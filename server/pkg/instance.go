package llamactl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Instance struct {
	ID      uuid.UUID
	Options *InstanceOptions

	// Status
	Running bool

	// Output channels
	StdOutChan chan string // Channel for sending output messages
	StdErrChan chan string // Channel for sending error messages

	// internal
	cmd      *exec.Cmd          // Command to run the instance
	ctx      context.Context    // Context for managing the instance lifecycle
	cancel   context.CancelFunc // Function to cancel the context
	stdout   io.ReadCloser      // Standard output stream
	stderr   io.ReadCloser      // Standard error stream
	mu       sync.Mutex         // Mutex for synchronizing access to the instance
	restarts int                // Number of restarts
}

func NewInstance(id uuid.UUID, options *InstanceOptions) *Instance {
	return &Instance{
		ID:      id,
		Options: options,

		Running: false,

		StdOutChan: make(chan string, 100),
		StdErrChan: make(chan string, 100),
	}
}

func (i *Instance) Start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Running {
		return fmt.Errorf("instance %s is already running", i.ID)
	}

	args := i.Options.BuildCommandArgs()

	i.ctx, i.cancel = context.WithCancel(context.Background())
	i.cmd = exec.CommandContext(i.ctx, "llama-server", args...)

	var err error
	i.stdout, err = i.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	i.stderr, err = i.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := i.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start instance %s: %w", i.ID, err)
	}

	i.Running = true

	go i.readOutput(i.stdout, i.StdOutChan, "stdout")
	go i.readOutput(i.stderr, i.StdErrChan, "stderr")

	go i.monitorProcess()

	return nil
}

// Stop terminates the subprocess
func (i *Instance) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.Running {
		return fmt.Errorf("instance %s is not running", i.ID)
	}

	// Cancel the context to signal termination
	i.cancel()

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- i.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited normally
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't exit within 5 seconds
		if i.cmd.Process != nil {
			i.cmd.Process.Kill()
		}
	}

	i.Running = false

	// Close channels when process is stopped
	close(i.StdOutChan)
	close(i.StdErrChan)

	return nil
}

// readOutput reads from the given reader and sends lines to the channel
func (i *Instance) readOutput(reader io.ReadCloser, ch chan string, streamType string) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case ch <- line:
		default:
			// Channel is full, drop the line
			log.Printf("Dropped %s line for instance %s: %s", streamType, i.ID, line)
		}
	}
}

func (i *Instance) monitorProcess() {
	err := i.cmd.Wait()

	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.Running {
		return
	}

	i.Running = false

	// Log the exit
	if err != nil {
		log.Printf("Instance %s crashed with error: %v", i.ID, err)
	} else {
		log.Printf("Instance %s exited cleanly", i.ID)
	}

	// Handle restart if process crashed and auto-restart is enabled
	if err != nil && i.Options.AutoRestart && i.restarts < i.Options.MaxRestarts {
		i.restarts++
		log.Printf("Auto-restarting instance %s (attempt %d/%d) in %v",
			i.ID, i.restarts, i.Options.MaxRestarts, i.Options.RestartDelay)

		// Unlock mutex during sleep to avoid blocking other operations
		i.mu.Unlock()
		time.Sleep(i.Options.RestartDelay)
		i.mu.Lock()

		// Attempt restart
		if err := i.Start(); err != nil {
			log.Printf("Failed to restart instance %s: %v", i.ID, err)
		} else {
			log.Printf("Successfully restarted instance %s", i.ID)
			i.restarts = 0 // Reset restart count on successful restart
		}
	} else if i.restarts >= i.Options.MaxRestarts {
		log.Printf("Instance %s exceeded max restart attempts (%d)", i.ID, i.Options.MaxRestarts)
	}
}
