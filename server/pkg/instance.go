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
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type CreateInstanceOptions struct {
	// Auto restart
	AutoRestart *bool `json:"auto_restart,omitempty"`
	MaxRestarts *int  `json:"max_restarts,omitempty"`
	// RestartDelay duration in seconds
	RestartDelay *int `json:"restart_delay_seconds,omitempty"`

	LlamaServerOptions `json:",inline"`
}

// Instance represents a running instance of the llama server
type Instance struct {
	Name           string                 `json:"name"`
	options        *CreateInstanceOptions `json:"-"`
	globalSettings *InstancesConfig

	// Status
	Running bool `json:"running"`

	// Log file
	logFile *os.File `json:"-"`

	// internal
	cmd      *exec.Cmd              `json:"-"` // Command to run the instance
	ctx      context.Context        `json:"-"` // Context for managing the instance lifecycle
	cancel   context.CancelFunc     `json:"-"` // Function to cancel the context
	stdout   io.ReadCloser          `json:"-"` // Standard output stream
	stderr   io.ReadCloser          `json:"-"` // Standard error stream
	mu       sync.RWMutex           `json:"-"` // RWMutex for better read/write separation
	restarts int                    `json:"-"` // Number of restarts
	proxy    *httputil.ReverseProxy `json:"-"` // Reverse proxy for this instance

	// Restart control
	restartCancel context.CancelFunc `json:"-"` // Cancel function for pending restarts
}

// NewInstance creates a new instance with the given name, log path, and options
func NewInstance(name string, globalSettings *InstancesConfig, options *CreateInstanceOptions) *Instance {
	// Make a deep copy of options to avoid modifying the original and to prevent data races
	optionsCopy := &CreateInstanceOptions{}
	if options != nil {
		// Copy the embedded LlamaServerOptions
		optionsCopy.LlamaServerOptions = options.LlamaServerOptions

		// Copy and validate pointer fields
		if options.AutoRestart != nil {
			autoRestart := *options.AutoRestart
			optionsCopy.AutoRestart = &autoRestart
		}
		if options.MaxRestarts != nil {
			maxRestarts := *options.MaxRestarts
			if maxRestarts > 100 {
				log.Printf("Instance %s MaxRestarts value (%d) limited to 100", name, maxRestarts)
				maxRestarts = 100
			} else if maxRestarts < 0 {
				log.Printf("Instance %s MaxRestarts value (%d) cannot be negative, setting to 0", name, maxRestarts)
				maxRestarts = 0
			}
			optionsCopy.MaxRestarts = &maxRestarts
		}
		if options.RestartDelay != nil {
			restartDelay := *options.RestartDelay
			if restartDelay < 1 {
				log.Printf("Instance %s RestartDelay value (%d) too low, setting to 5 seconds", name, restartDelay)
				restartDelay = 5
			} else if restartDelay > 300 {
				log.Printf("Instance %s RestartDelay value (%d) too high, limiting to 300 seconds", name, restartDelay)
				restartDelay = 300
			}
			optionsCopy.RestartDelay = &restartDelay
		}
	}

	// Set defaults for restart options if not provided
	if optionsCopy.AutoRestart == nil {
		defaultAutoRestart := globalSettings.DefaultAutoRestart
		optionsCopy.AutoRestart = &defaultAutoRestart
	}
	if optionsCopy.MaxRestarts == nil {
		defaultMaxRestarts := globalSettings.DefaultMaxRestarts
		optionsCopy.MaxRestarts = &defaultMaxRestarts
	}
	if optionsCopy.RestartDelay == nil {
		defaultRestartDelay := globalSettings.DefaultRestartDelay
		optionsCopy.RestartDelay = &defaultRestartDelay
	}

	return &Instance{
		Name:           name,
		options:        optionsCopy,
		globalSettings: globalSettings,

		Running: false,
	}
}

// createLogFile creates and opens the log files for stdout and stderr
func (i *Instance) createLogFile() error {
	logPath := i.globalSettings.LogDirectory + "/" + i.Name + ".log"
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create stdout log file: %w", err)
	}

	i.logFile = logFile

	// Write a startup marker to both files
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(i.logFile, "\n=== Instance %s started at %s ===\n", i.Name, timestamp)

	return nil
}

// closeLogFile closes the log files
func (i *Instance) closeLogFile() {
	if i.logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(i.logFile, "=== Instance %s stopped at %s ===\n\n", i.Name, timestamp)
		i.logFile.Close()
		i.logFile = nil
	}
}

func (i *Instance) GetOptions() *CreateInstanceOptions {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.options
}

func (i *Instance) SetOptions(options *CreateInstanceOptions) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if options == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}

	// Make a deep copy to avoid sharing references
	optionsCopy := &CreateInstanceOptions{}
	optionsCopy.LlamaServerOptions = options.LlamaServerOptions

	// Copy pointer fields (validation already done in UnmarshalJSON)
	if options.AutoRestart != nil {
		autoRestart := *options.AutoRestart
		optionsCopy.AutoRestart = &autoRestart
	}
	if options.MaxRestarts != nil {
		maxRestarts := *options.MaxRestarts
		optionsCopy.MaxRestarts = &maxRestarts
	}
	if options.RestartDelay != nil {
		restartDelay := *options.RestartDelay
		optionsCopy.RestartDelay = &restartDelay
	}

	i.options = optionsCopy
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

	// Safety check: ensure options are valid
	if i.options == nil {
		return fmt.Errorf("instance %s has no options set", i.Name)
	}

	// Create log files
	if err := i.createLogFile(); err != nil {
		return fmt.Errorf("failed to create log files: %w", err)
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
		i.closeLogFile() // Ensure log files are closed on error
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	i.stderr, err = i.cmd.StderrPipe()
	if err != nil {
		i.stdout.Close() // Ensure stdout is closed on error
		i.closeLogFile() // Ensure log files are closed on error
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := i.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start instance %s: %w", i.Name, err)
	}

	i.Running = true

	go i.readOutput(i.stdout, i.logFile)
	go i.readOutput(i.stderr, i.logFile)

	go i.monitorProcess()

	return nil
}

// Stop terminates the subprocess
func (i *Instance) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.Running {
		// Even if not running, cancel any pending restart
		if i.restartCancel != nil {
			i.restartCancel()
			i.restartCancel = nil
			log.Printf("Cancelled pending restart for instance %s", i.Name)
		}
		return fmt.Errorf("instance %s is not running", i.Name)
	}

	// Cancel any pending restart
	if i.restartCancel != nil {
		i.restartCancel()
		i.restartCancel = nil
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

	i.closeLogFile() // Close log files after stopping

	return nil
}

// GetLogs retrieves the last n lines of logs from the instance
func (i *Instance) GetLogs(num_lines int) (string, error) {
	i.mu.RLock()
	logFileName := ""
	if i.logFile != nil {
		logFileName = i.logFile.Name()
	}
	i.mu.RUnlock()

	if logFileName == "" {
		return "", fmt.Errorf("log file not created for instance %s", i.Name)
	}

	file, err := os.Open(logFileName)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	if num_lines <= 0 {
		content, err := io.ReadAll(file)
		if err != nil {
			return "", fmt.Errorf("failed to read log file: %w", err)
		}
		return string(content), nil
	}

	var lines []string
	scanner := bufio.NewScanner(file)

	// Read all lines into a slice
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// Return the last N lines
	start := max(len(lines)-num_lines, 0)

	return strings.Join(lines[start:], "\n"), nil
}

// readOutput reads from the given reader and writes lines to the log file
func (i *Instance) readOutput(reader io.ReadCloser, logFile *os.File) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if logFile != nil {
			fmt.Fprintln(logFile, line)
			logFile.Sync() // Ensure data is written to disk
		}
	}
}

func (i *Instance) monitorProcess() {
	err := i.cmd.Wait()

	i.mu.Lock()

	if !i.Running {
		i.mu.Unlock()
		return
	}

	i.Running = false
	i.closeLogFile()

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
func (i *Instance) handleRestart() {
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
func (i *Instance) validateRestartConditions() (shouldRestart bool, maxRestarts int, restartDelay int) {
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

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	// Use read lock since we're only reading data
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Create a temporary struct with exported fields for JSON marshalling
	temp := struct {
		Name    string                 `json:"name"`
		Options *CreateInstanceOptions `json:"options,omitempty"`
		Running bool                   `json:"running"`
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
		Name    string                 `json:"name"`
		Options *CreateInstanceOptions `json:"options,omitempty"`
		Running bool                   `json:"running"`
	}{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the fields
	i.Name = temp.Name
	i.Running = temp.Running

	// Handle options with validation
	if temp.Options != nil {
		// Validate and sanitize restart parameters
		if temp.Options.MaxRestarts != nil {
			if *temp.Options.MaxRestarts < 0 {
				log.Printf("Instance %s MaxRestarts value (%d) cannot be negative, setting to 0", i.Name, *temp.Options.MaxRestarts)
				maxRestarts := 0
				temp.Options.MaxRestarts = &maxRestarts
			}
		}

		if temp.Options.RestartDelay != nil {
			if *temp.Options.RestartDelay < 0 {
				log.Printf("Instance %s RestartDelay value (%d) cannot be negative, setting to 0", i.Name, *temp.Options.RestartDelay)
				restartDelay := 5
				temp.Options.RestartDelay = &restartDelay
			}
		}

		i.options = temp.Options
	}

	return nil
}
