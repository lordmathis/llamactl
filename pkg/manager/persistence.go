package manager

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/instance"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// instancePersister provides atomic file-based persistence with durability guarantees.
type instancePersister struct {
	mu           sync.Mutex
	instancesDir string
	enabled      bool
}

// newInstancePersister creates a new instance persister.
// If instancesDir is empty, persistence is disabled.
func newInstancePersister(instancesDir string) (*instancePersister, error) {
	if instancesDir == "" {
		return &instancePersister{
			enabled: false,
		}, nil
	}

	// Ensure the instances directory exists
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instances directory: %w", err)
	}

	return &instancePersister{
		instancesDir: instancesDir,
		enabled:      true,
	}, nil
}

// Save persists an instance to disk with atomic write
func (p *instancePersister) save(inst *instance.Instance) error {
	if !p.enabled {
		return nil
	}

	if inst == nil {
		return fmt.Errorf("cannot save nil instance")
	}

	// Validate instance name to prevent path traversal
	if err := p.validateInstanceName(inst.Name); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	instancePath := filepath.Join(p.instancesDir, inst.Name+".json")
	tempPath := instancePath + ".tmp"

	// Serialize instance to JSON
	jsonData, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance %s: %w", inst.Name, err)
	}

	// Create temporary file
	tempFile, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file for instance %s: %w", inst.Name, err)
	}

	// Write data to temporary file
	if _, err := tempFile.Write(jsonData); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to write temp file for instance %s: %w", inst.Name, err)
	}

	// Sync to disk before rename to ensure durability
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync temp file for instance %s: %w", inst.Name, err)
	}

	// Close the file
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file for instance %s: %w", inst.Name, err)
	}

	// Atomic rename (this is atomic on POSIX systems)
	if err := os.Rename(tempPath, instancePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file for instance %s: %w", inst.Name, err)
	}

	return nil
}

// Delete removes an instance's persistence file from disk.
func (p *instancePersister) delete(name string) error {
	if !p.enabled {
		return nil
	}

	if err := p.validateInstanceName(name); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	instancePath := filepath.Join(p.instancesDir, name+".json")

	if err := os.Remove(instancePath); err != nil {
		if os.IsNotExist(err) {
			// Not an error if file doesn't exist
			return nil
		}
		return fmt.Errorf("failed to delete instance file for %s: %w", name, err)
	}

	return nil
}

// LoadAll loads all persisted instances from disk.
// Returns a slice of instances and any errors encountered during loading.
func (p *instancePersister) loadAll() ([]*instance.Instance, error) {
	if !p.enabled {
		return nil, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if instances directory exists
	if _, err := os.Stat(p.instancesDir); os.IsNotExist(err) {
		return nil, nil // No instances directory, return empty list
	}

	// Read all JSON files from instances directory
	files, err := os.ReadDir(p.instancesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read instances directory: %w", err)
	}

	instances := make([]*instance.Instance, 0)
	var loadErrors []string

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		instanceName := strings.TrimSuffix(file.Name(), ".json")
		instancePath := filepath.Join(p.instancesDir, file.Name())

		inst, err := p.loadInstanceFile(instanceName, instancePath)
		if err != nil {
			log.Printf("Failed to load instance %s: %v", instanceName, err)
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", instanceName, err))
			continue
		}

		instances = append(instances, inst)
	}

	if len(loadErrors) > 0 {
		log.Printf("Loaded %d instances with %d errors", len(instances), len(loadErrors))
	} else if len(instances) > 0 {
		log.Printf("Loaded %d instances from persistence", len(instances))
	}

	return instances, nil
}

// loadInstanceFile is an internal helper that loads a single instance file.
// Note: This assumes the mutex is already held by the caller.
func (p *instancePersister) loadInstanceFile(name, path string) (*instance.Instance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance file: %w", err)
	}

	var inst instance.Instance
	if err := json.Unmarshal(data, &inst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	// Validate the instance name matches the filename
	if inst.Name != name {
		return nil, fmt.Errorf("instance name mismatch: file=%s, instance.Name=%s", name, inst.Name)
	}

	return &inst, nil
}

// validateInstanceName ensures the instance name is safe for filesystem operations.
func (p *instancePersister) validateInstanceName(name string) error {
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	cleaned := filepath.Clean(name)

	// After cleaning, name should not contain any path separators
	if cleaned != name || strings.Contains(cleaned, string(filepath.Separator)) {
		return fmt.Errorf("invalid instance name: %s", name)
	}

	return nil
}
