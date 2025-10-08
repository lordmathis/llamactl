package manager

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"log"
	"net/http"
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
	IsMaxRunningInstancesReached() bool
	StopInstance(name string) (*instance.Process, error)
	EvictLRUInstance() error
	RestartInstance(name string) (*instance.Process, error)
	GetInstanceLogs(name string) (string, error)
	Shutdown()
}

type RemoteManager interface {
	ListRemoteInstances(node *config.NodeConfig) ([]*instance.Process, error)
	CreateRemoteInstance(node *config.NodeConfig, name string, options *instance.CreateInstanceOptions) (*instance.Process, error)
	GetRemoteInstance(node *config.NodeConfig, name string) (*instance.Process, error)
	UpdateRemoteInstance(node *config.NodeConfig, name string, options *instance.CreateInstanceOptions) (*instance.Process, error)
	DeleteRemoteInstance(node *config.NodeConfig, name string) error
	StartRemoteInstance(node *config.NodeConfig, name string) (*instance.Process, error)
	StopRemoteInstance(node *config.NodeConfig, name string) (*instance.Process, error)
	RestartRemoteInstance(node *config.NodeConfig, name string) (*instance.Process, error)
	GetRemoteInstanceLogs(node *config.NodeConfig, name string) (string, error)
}

type instanceManager struct {
	mu               sync.RWMutex
	instances        map[string]*instance.Process
	runningInstances map[string]struct{}
	ports            map[int]bool
	instancesConfig  config.InstancesConfig
	backendsConfig   config.BackendConfig

	// Timeout checker
	timeoutChecker *time.Ticker
	shutdownChan   chan struct{}
	shutdownDone   chan struct{}
	isShutdown     bool

	// Remote instance management
	httpClient        *http.Client
	instanceNodeMap   map[string]*config.NodeConfig // Maps instance name to its node config
	nodeConfigMap     map[string]*config.NodeConfig // Maps node name to node config for quick lookup
}

// NewInstanceManager creates a new instance of InstanceManager.
func NewInstanceManager(backendsConfig config.BackendConfig, instancesConfig config.InstancesConfig, nodesConfig map[string]config.NodeConfig) InstanceManager {
	if instancesConfig.TimeoutCheckInterval <= 0 {
		instancesConfig.TimeoutCheckInterval = 5 // Default to 5 minutes if not set
	}

	// Build node config map for quick lookup
	nodeConfigMap := make(map[string]*config.NodeConfig)
	for name := range nodesConfig {
		nodeCopy := nodesConfig[name]
		nodeConfigMap[name] = &nodeCopy
	}

	im := &instanceManager{
		instances:        make(map[string]*instance.Process),
		runningInstances: make(map[string]struct{}),
		ports:            make(map[int]bool),
		instancesConfig:  instancesConfig,
		backendsConfig:   backendsConfig,

		timeoutChecker: time.NewTicker(time.Duration(instancesConfig.TimeoutCheckInterval) * time.Minute),
		shutdownChan:   make(chan struct{}),
		shutdownDone:   make(chan struct{}),

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},

		instanceNodeMap: make(map[string]*config.NodeConfig),
		nodeConfigMap:   nodeConfigMap,
	}

	// Load existing instances from disk
	if err := im.loadInstances(); err != nil {
		log.Printf("Error loading instances: %v", err)
	}

	// Start the timeout checker goroutine after initialization is complete
	go func() {
		defer close(im.shutdownDone)

		for {
			select {
			case <-im.timeoutChecker.C:
				im.checkAllTimeouts()
			case <-im.shutdownChan:
				return // Exit goroutine on shutdown
			}
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

	// Check if already shutdown
	if im.isShutdown {
		im.mu.Unlock()
		return
	}
	im.isShutdown = true

	// Signal the timeout checker to stop
	close(im.shutdownChan)

	// Create a list of running instances to stop
	var runningInstances []*instance.Process
	var runningNames []string
	for name, inst := range im.instances {
		if inst.IsRunning() {
			runningInstances = append(runningInstances, inst)
			runningNames = append(runningNames, name)
		}
	}

	// Release lock before stopping instances to avoid deadlock
	im.mu.Unlock()

	// Wait for the timeout checker goroutine to actually stop
	<-im.shutdownDone

	// Now stop the ticker
	if im.timeoutChecker != nil {
		im.timeoutChecker.Stop()
	}

	// Stop instances without holding the manager lock
	var wg sync.WaitGroup
	wg.Add(len(runningInstances))

	for i, inst := range runningInstances {
		go func(name string, inst *instance.Process) {
			defer wg.Done()
			fmt.Printf("Stopping instance %s...\n", name)
			// Attempt to stop the instance gracefully
			if err := inst.Stop(); err != nil {
				fmt.Printf("Error stopping instance %s: %v\n", name, err)
			}
		}(runningNames[i], inst)
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

	options := persistedInstance.GetOptions()

	// Check if this is a remote instance
	isRemote := options != nil && len(options.Nodes) > 0

	var statusCallback func(oldStatus, newStatus instance.InstanceStatus)
	if !isRemote {
		// Only set status callback for local instances
		statusCallback = func(oldStatus, newStatus instance.InstanceStatus) {
			im.onStatusChange(persistedInstance.Name, oldStatus, newStatus)
		}
	}

	// Create new inst using NewInstance (handles validation, defaults, setup)
	inst := instance.NewInstance(name, &im.backendsConfig, &im.instancesConfig, options, statusCallback)

	// Restore persisted fields that NewInstance doesn't set
	inst.Created = persistedInstance.Created
	inst.SetStatus(persistedInstance.Status)

	// Handle remote instance mapping
	if isRemote {
		nodeName := options.Nodes[0]
		nodeConfig, exists := im.nodeConfigMap[nodeName]
		if !exists {
			return fmt.Errorf("node %s not found for remote instance %s", nodeName, name)
		}
		im.instanceNodeMap[name] = nodeConfig
	} else {
		// Check for port conflicts only for local instances
		if inst.GetPort() > 0 {
			port := inst.GetPort()
			if im.ports[port] {
				return fmt.Errorf("port conflict: instance %s wants port %d which is already in use", name, port)
			}
			im.ports[port] = true
		}
	}

	im.instances[name] = inst
	return nil
}

// autoStartInstances starts instances that were running when persisted and have auto-restart enabled
// For instances with auto-restart disabled, it sets their status to Stopped
func (im *instanceManager) autoStartInstances() {
	im.mu.RLock()
	var instancesToStart []*instance.Process
	var instancesToStop []*instance.Process
	for _, inst := range im.instances {
		if inst.IsRunning() && // Was running when persisted
			inst.GetOptions() != nil &&
			inst.GetOptions().AutoRestart != nil {
			if *inst.GetOptions().AutoRestart {
				instancesToStart = append(instancesToStart, inst)
			} else {
				// Instance was running but auto-restart is disabled, mark as stopped
				instancesToStop = append(instancesToStop, inst)
			}
		}
	}
	im.mu.RUnlock()

	// Stop instances that have auto-restart disabled
	for _, inst := range instancesToStop {
		log.Printf("Instance %s was running but auto-restart is disabled, setting status to stopped", inst.Name)
		inst.SetStatus(instance.Stopped)
	}

	// Start instances that have auto-restart enabled
	for _, inst := range instancesToStart {
		log.Printf("Auto-starting instance %s", inst.Name)
		// Reset running state before starting (since Start() expects stopped instance)
		inst.SetStatus(instance.Stopped)
		if err := inst.Start(); err != nil {
			log.Printf("Failed to auto-start instance %s: %v", inst.Name, err)
		}
	}
}

func (im *instanceManager) onStatusChange(name string, oldStatus, newStatus instance.InstanceStatus) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if newStatus == instance.Running {
		im.runningInstances[name] = struct{}{}
	} else {
		delete(im.runningInstances, name)
	}
}

// getNodeForInstance returns the node configuration for a remote instance
// Returns nil if the instance is not remote or the node is not found
func (im *instanceManager) getNodeForInstance(inst *instance.Process) *config.NodeConfig {
	if !inst.IsRemote() {
		return nil
	}

	// Check if we have a cached mapping
	if nodeConfig, exists := im.instanceNodeMap[inst.Name]; exists {
		return nodeConfig
	}

	return nil
}
