package instance

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type logger struct {
	name        string
	logDir      string
	logFile     atomic.Pointer[os.File]
	logFilePath string
	mu          sync.RWMutex
}

func newLogger(name string, logDir string) *logger {
	return &logger{
		name:   name,
		logDir: logDir,
	}
}

// create creates and opens the log files for stdout and stderr
func (i *logger) create() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.logDir == "" {
		return fmt.Errorf("logDir is empty for instance %s", i.name)
	}

	// Set up instance logs
	logPath := i.logDir + "/" + i.name + ".log"

	i.logFilePath = logPath
	if err := os.MkdirAll(i.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create stdout log file: %w", err)
	}

	i.logFile.Store(logFile)

	// Write a startup marker to both files
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(logFile, "\n=== Instance %s started at %s ===\n", i.name, timestamp)

	return nil
}

// getLogs retrieves the last n lines of logs from the instance
func (i *logger) getLogs(num_lines int) (string, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.logFilePath == "" {
		return "", fmt.Errorf("log file not created for instance %s", i.name)
	}

	file, err := os.Open(i.logFilePath)
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

// close closes the log files
func (i *logger) close() {
	i.mu.Lock()
	defer i.mu.Unlock()

	logFile := i.logFile.Swap(nil)
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(logFile, "=== Instance %s stopped at %s ===\n\n", i.name, timestamp)
		logFile.Sync() // Ensure all buffered data is written to disk
		logFile.Close()
	}
}

// readOutput reads from the given reader and writes lines to the log file
func (i *logger) readOutput(reader io.ReadCloser) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Use atomic load to avoid lock contention on every line
		if logFile := i.logFile.Load(); logFile != nil {
			fmt.Fprintln(logFile, line)
		}
	}
}
