package llamactl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Instance struct {
	Name    string           `json:"name"`
	options *InstanceOptions `json:"-"` // Now unexported - access via GetOptions/SetOptions

	// Status
	Running bool `json:"running"`

	// Output channels
	StdOutChan chan string `json:"-"` // Channel for sending output messages
	StdErrChan chan string `json:"-"` // Channel for sending error messages

	// internal
	cmd      *exec.Cmd              `json:"-"` // Command to run the instance
	ctx      context.Context        `json:"-"` // Context for managing the instance lifecycle
	cancel   context.CancelFunc     `json:"-"` // Function to cancel the context
	stdout   io.ReadCloser          `json:"-"` // Standard output stream
	stderr   io.ReadCloser          `json:"-"` // Standard error stream
	mu       sync.Mutex             `json:"-"` // Mutex for synchronizing access to the instance
	restarts int                    `json:"-"` // Number of restarts
	proxy    *httputil.ReverseProxy `json:"-"` // Reverse proxy for this instance
}

func NewInstance(name string, options *InstanceOptions) *Instance {
	return &Instance{
		Name:    name,
		options: options,

		Running: false,

		StdOutChan: make(chan string, 100),
		StdErrChan: make(chan string, 100),
	}
}

func (i *Instance) GetOptions() *InstanceOptions {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.options
}

func (i *Instance) SetOptions(options *InstanceOptions) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if options == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}
	i.options = options
	// Clear the proxy so it gets recreated with new options
	i.proxy = nil
}

// GetProxy returns the reverse proxy for this instance, creating it if needed
func (i *Instance) GetProxy() (*httputil.ReverseProxy, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.proxy == nil {
		if i.options == nil {
			return nil, fmt.Errorf("instance %s has no options set", i.Name)
		}

		targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", i.options.Host, i.options.Port))
		if err != nil {
			return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", i.Name, err)
		}

		i.proxy = httputil.NewSingleHostReverseProxy(targetURL)
	}

	return i.proxy, nil
}

func (i *Instance) Start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Running {
		return fmt.Errorf("instance %s is already running", i.Name)
	}

	args := i.options.BuildCommandArgs()

	i.ctx, i.cancel = context.WithCancel(context.Background())
	i.cmd = exec.CommandContext(i.ctx, "llama-server", args...)

	if runtime.GOOS != "windows" {
		if i.cmd.SysProcAttr == nil {
			i.cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		i.cmd.SysProcAttr.Setpgid = true
	}

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
		return fmt.Errorf("failed to start instance %s: %w", i.Name, err)
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
		return fmt.Errorf("instance %s is not running", i.Name)
	}

	// Cancel the context to signal termination
	i.cancel()

	// Clean up the proxy
	i.proxy = nil

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
			log.Printf("Dropped %s line for instance %s: %s", streamType, i.Name, line)
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
		log.Printf("Instance %s crashed with error: %v", i.Name, err)
	} else {
		log.Printf("Instance %s exited cleanly", i.Name)
	}

	// Handle restart if process crashed and auto-restart is enabled
	if err != nil && i.options.AutoRestart && i.restarts < i.options.MaxRestarts {
		i.restarts++
		log.Printf("Auto-restarting instance %s (attempt %d/%d) in %v",
			i.Name, i.restarts, i.options.MaxRestarts, i.options.RestartDelay.ToDuration())

		// Unlock mutex during sleep to avoid blocking other operations
		i.mu.Unlock()
		time.Sleep(i.options.RestartDelay.ToDuration())
		i.mu.Lock()

		// Attempt restart
		if err := i.Start(); err != nil {
			log.Printf("Failed to restart instance %s: %v", i.Name, err)
		} else {
			log.Printf("Successfully restarted instance %s", i.Name)
			i.restarts = 0 // Reset restart count on successful restart
		}
	} else if i.restarts >= i.options.MaxRestarts {
		log.Printf("Instance %s exceeded max restart attempts (%d)", i.Name, i.options.MaxRestarts)
	}
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Create a temporary struct with exported fields for JSON marshalling
	temp := struct {
		Name    string           `json:"name"`
		Options *InstanceOptions `json:"options,omitempty"`
		Running bool             `json:"running"`
	}{
		Name:    i.Name,
		Options: i.options,
		Running: i.Running,
	}

	return json.Marshal(temp)
}

// UnmarshalJSON implements json.Unmarshaler for Instance
func (i *Instance) UnmarshalJSON(data []byte) error {
	// Create a temporary struct for unmarshalling
	temp := struct {
		Name    string           `json:"name"`
		Options *InstanceOptions `json:"options,omitempty"`
		Running bool             `json:"running"`
	}{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the fields
	i.Name = temp.Name
	i.Running = temp.Running

	// Handle options - ensure embedded LlamaServerOptions is initialized
	if temp.Options != nil {
		i.options = temp.Options
	}

	// Initialize channels if they don't exist
	if i.StdOutChan == nil {
		i.StdOutChan = make(chan string, 100)
	}
	if i.StdErrChan == nil {
		i.StdErrChan = make(chan string, 100)
	}

	return nil
}
