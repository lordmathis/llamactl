package instance

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	timber "github.com/DeRuina/timberjack"
	"llamactl/pkg/config"
)

type logger struct {
	name        string
	logDir      string
	logFile     *timber.Logger
	logFilePath string
	mu          sync.RWMutex
	cfg         *config.LogRotationConfig
}

func newLogger(name, logDir string, cfg *config.LogRotationConfig) *logger {
	return &logger{
		name:   name,
		logDir: logDir,
		cfg:    cfg,
	}
}

func (l *logger) create() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logDir == "" {
		return fmt.Errorf("logDir empty for instance %s", l.name)
	}

	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := fmt.Sprintf("%s/%s.log", l.logDir, l.name)
	l.logFilePath = logPath

	// Build the timber logger
	t := &timber.Logger{
		Filename:   logPath,
		MaxSize:    l.cfg.MaxSizeMB,
		MaxBackups: 0, // No limit on backups - use index-based naming
		// Compression: "gzip" if Compress is true, else "none"
		Compression: func() string {
			if l.cfg.Compress {
				return "gzip"
			}
			return "none"
		}(),
		FileMode:  0644,  // default; timberjack uses 640 if 0
		LocalTime: false, // Use index-based naming instead of timestamps
	}

	// If rotation is disabled, set MaxSize to 0 so no rotation occurs
	if !l.cfg.Enabled {
		t.MaxSize = 0
	}

	l.logFile = t

	// Write a startup marker
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(t, "\n=== Instance %s started at %s ===\n", l.name, ts)

	return nil
}

func (l *logger) readOutput(rc io.ReadCloser) {
	defer rc.Close()
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		if lg := l.logFile; lg != nil {
			fmt.Fprintln(lg, line) // timber.Logger implements io.Writer
		}
	}
}

func (l *logger) close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	lg := l.logFile
	if lg == nil {
		return
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(lg, "=== Instance %s stopped at %s ===\n\n", l.name, ts)

	_ = lg.Close() // shuts down any background goroutines (none in this config)
	l.logFile = nil
}

// getLogs retrieves the last n lines of logs from the instance
func (l *logger) getLogs(num_lines int) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logFilePath == "" {
		return "", fmt.Errorf("log file not created for instance %s", l.name)
	}

	file, err := os.Open(l.logFilePath)
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
