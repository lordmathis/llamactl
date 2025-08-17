package manager

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// InstanceManager defines the interface for managing instances of the llama server.
type InstanceManager interface {
	ListInstances() ([]*instance.Process, error)
	CreateInstance(name string, options *instance.CreateInstanceOptions) (*instance.Process, error)
	GetInstance(name string) (*instance.Process, error)
	UpdateInstance(name string, options *instance.CreateInstanceOptions) (*instance.Process, error)
	DeleteInstance(name string) error
	StartInstance(name string) (*instance.Process, error)
	StopInstance(name string) (*instance.Process, error)
	RestartInstance(name string) (*instance.Process, error)
	GetInstanceLogs(name string) (string, error)
	Shutdown()
}

type instanceManager struct {
	mu              sync.RWMutex
	instances       map[string]*instance.Process
	ports           map[int]bool
	instancesConfig config.InstancesConfig

	// Timeout checker
	timeoutChecker *time.Ticker
}

// NewInstanceManager creates a new instance of InstanceManager.
func NewInstanceManager(instancesConfig config.InstancesConfig) InstanceManager {
	im := &instanceManager{
		instances:       make(map[string]*instance.Process),
		ports:           make(map[int]bool),
		instancesConfig: instancesConfig,

		timeoutChecker: time.NewTicker(time.Duration(instancesConfig.TimeoutCheckInterval) * time.Minute),
	}

	// Load existing instances from disk
	if err := im.loadInstances(); err != nil {
		log.Printf("Error loading instances: %v", err)
	}

	go func() {
		for range im.timeoutChecker.C {
			im.checkAllTimeouts()
		}
	}()

	return im
}

func (im *instanceManager) getNextAvailablePort() (int, error) {
	portRange := im.instancesConfig.PortRange

	for port := portRange[0]; port <= portRange[1]; port++ {
		if !im.ports[port] {
			im.ports[port] = true
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in the specified range")
}

// persistInstance saves an instance to its JSON file
func (im *instanceManager) persistInstance(instance *instance.Process) error {
	if im.instancesConfig.InstancesDir == "" {
		return nil // Persistence disabled
	}

	instancePath := filepath.Join(im.instancesConfig.InstancesDir, instance.Name+".json")
	tempPath := instancePath + ".tmp"

	// Serialize instance to JSON
	jsonData, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance %s: %w", instance.Name, err)
	}

	// Write to temporary file first
	if err := os.WriteFile(tempPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file for instance %s: %w", instance.Name, err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, instancePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file for instance %s: %w", instance.Name, err)
	}

	return nil
}

func (im *instanceManager) Shutdown() {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Stop the timeout checker
	if im.timeoutChecker != nil {
		im.timeoutChecker.Stop()
		im.timeoutChecker = nil
	}

	var wg sync.WaitGroup
	wg.Add(len(im.instances))

	for name, inst := range im.instances {
		if !inst.Running {
			wg.Done() // If instance is not running, just mark it as done
			continue
		}

		go func(name string, inst *instance.Process) {
			defer wg.Done()
			fmt.Printf("Stopping instance %s...\n", name)
			// Attempt to stop the instance gracefully
			if err := inst.Stop(); err != nil {
				fmt.Printf("Error stopping instance %s: %v\n", name, err)
			}
		}(name, inst)
	}

	wg.Wait()
	fmt.Println("All instances stopped.")
}

// loadInstances restores all instances from disk
func (im *instanceManager) loadInstances() error {
	if im.instancesConfig.InstancesDir == "" {
		return nil // Persistence disabled
	}

	// Check if instances directory exists
	if _, err := os.Stat(im.instancesConfig.InstancesDir); os.IsNotExist(err) {
		return nil // No instances directory, start fresh
	}

	// Read all JSON files from instances directory
	files, err := os.ReadDir(im.instancesConfig.InstancesDir)
	if err != nil {
		return fmt.Errorf("failed to read instances directory: %w", err)
	}

	loadedCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		instanceName := strings.TrimSuffix(file.Name(), ".json")
		instancePath := filepath.Join(im.instancesConfig.InstancesDir, file.Name())

		if err := im.loadInstance(instanceName, instancePath); err != nil {
			log.Printf("Failed to load instance %s: %v", instanceName, err)
			continue
		}

		loadedCount++
	}

	if loadedCount > 0 {
		log.Printf("Loaded %d instances from persistence", loadedCount)
		// Auto-start instances that have auto-restart enabled
		go im.autoStartInstances()
	}

	return nil
}

// loadInstance loads a single instance from its JSON file
func (im *instanceManager) loadInstance(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read instance file: %w", err)
	}

	var persistedInstance instance.Process
	if err := json.Unmarshal(data, &persistedInstance); err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	// Validate the instance name matches the filename
	if persistedInstance.Name != name {
		return fmt.Errorf("instance name mismatch: file=%s, instance.Name=%s", name, persistedInstance.Name)
	}

	// Create new inst using NewInstance (handles validation, defaults, setup)
	inst := instance.NewInstance(name, &im.instancesConfig, persistedInstance.GetOptions())

	// Restore persisted fields that NewInstance doesn't set
	inst.Created = persistedInstance.Created
	inst.Running = persistedInstance.Running

	// Check for port conflicts and add to maps
	if inst.GetOptions() != nil && inst.GetOptions().Port > 0 {
		port := inst.GetOptions().Port
		if im.ports[port] {
			return fmt.Errorf("port conflict: instance %s wants port %d which is already in use", name, port)
		}
		im.ports[port] = true
	}

	im.instances[name] = inst
	return nil
}

// autoStartInstances starts instances that were running when persisted and have auto-restart enabled
func (im *instanceManager) autoStartInstances() {
	im.mu.RLock()
	var instancesToStart []*instance.Process
	for _, inst := range im.instances {
		if inst.Running && // Was running when persisted
			inst.GetOptions() != nil &&
			inst.GetOptions().AutoRestart != nil &&
			*inst.GetOptions().AutoRestart {
			instancesToStart = append(instancesToStart, inst)
		}
	}
	im.mu.RUnlock()

	for _, inst := range instancesToStart {
		log.Printf("Auto-starting instance %s", inst.Name)
		// Reset running state before starting (since Start() expects stopped instance)
		inst.Running = false
		if err := inst.Start(); err != nil {
			log.Printf("Failed to auto-start instance %s: %v", inst.Name, err)
		}
	}
}
