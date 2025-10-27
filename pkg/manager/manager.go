package manager

import (
	"context"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"log"
	"sync"
	"time"
)

// InstanceManager defines the interface for managing instances of the llama server.
type InstanceManager interface {
	ListInstances() ([]*instance.Instance, error)
	CreateInstance(name string, options *instance.Options) (*instance.Instance, error)
	GetInstance(name string) (*instance.Instance, error)
	UpdateInstance(name string, options *instance.Options) (*instance.Instance, error)
	DeleteInstance(name string) error
	StartInstance(name string) (*instance.Instance, error)
	IsMaxRunningInstancesReached() bool
	StopInstance(name string) (*instance.Instance, error)
	EvictLRUInstance() error
	RestartInstance(name string) (*instance.Instance, error)
	GetInstanceLogs(name string, numLines int) (string, error)
	Shutdown()
}

type instanceManager struct {
	// Components (each with own synchronization)
	registry    *instanceRegistry
	ports       *portAllocator
	persistence *instancePersister
	remote      *remoteManager
	lifecycle   *lifecycleManager

	// Configuration
	globalConfig *config.AppConfig

	// Synchronization
	instanceLocks sync.Map // map[string]*sync.Mutex - per-instance locks for concurrent operations
	shutdownOnce  sync.Once
}

// New creates a new instance of InstanceManager.
func New(globalConfig *config.AppConfig) InstanceManager {

	if globalConfig.Instances.TimeoutCheckInterval <= 0 {
		globalConfig.Instances.TimeoutCheckInterval = 5 // Default to 5 minutes if not set
	}

	// Initialize components
	registry := newInstanceRegistry()

	// Initialize port allocator
	portRange := globalConfig.Instances.PortRange
	ports := newPortAllocator(portRange[0], portRange[1])

	// Initialize persistence
	persistence := newInstancePersister(globalConfig.Instances.InstancesDir)

	// Initialize remote manager
	remote := newRemoteManager(globalConfig.Nodes, 30*time.Second)

	// Create manager instance
	im := &instanceManager{
		registry:     registry,
		ports:        ports,
		persistence:  persistence,
		remote:       remote,
		globalConfig: globalConfig,
	}

	// Initialize lifecycle manager (needs reference to manager for Stop/Evict operations)
	checkInterval := time.Duration(globalConfig.Instances.TimeoutCheckInterval) * time.Minute
	im.lifecycle = newLifecycleManager(registry, im, checkInterval, true)

	// Load existing instances from disk
	if err := im.loadInstances(); err != nil {
		log.Printf("Error loading instances: %v", err)
	}

	// Start the lifecycle manager
	im.lifecycle.start()

	return im
}

// persistInstance saves an instance using the persistence component
func (im *instanceManager) persistInstance(inst *instance.Instance) error {
	return im.persistence.save(inst)
}

func (im *instanceManager) Shutdown() {
	im.shutdownOnce.Do(func() {
		// 1. Stop lifecycle manager (stops timeout checker)
		im.lifecycle.stop()

		// 2. Get running instances (no lock needed - registry handles it)
		running := im.registry.listRunning()

		// 3. Stop local instances concurrently
		var wg sync.WaitGroup
		for _, inst := range running {
			if inst.IsRemote() {
				continue // Skip remote instances
			}
			wg.Add(1)
			go func(inst *instance.Instance) {
				defer wg.Done()
				fmt.Printf("Stopping instance %s...\n", inst.Name)
				if err := inst.Stop(); err != nil {
					fmt.Printf("Error stopping instance %s: %w\n", inst.Name, err)
				}
			}(inst)
		}
		wg.Wait()
		fmt.Println("All instances stopped.")
	})
}

// loadInstances restores all instances from disk using the persistence component
func (im *instanceManager) loadInstances() error {
	// Load all instances from persistence
	instances, err := im.persistence.loadAll()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	if len(instances) == 0 {
		return nil
	}

	// Process each loaded instance
	for _, persistedInst := range instances {
		if err := im.loadInstance(persistedInst); err != nil {
			log.Printf("Failed to load instance %s: %v", persistedInst.Name, err)
			continue
		}
	}

	log.Printf("Loaded %d instances from persistence", len(instances))

	// Auto-start instances that have auto-restart enabled
	go im.autoStartInstances()

	return nil
}

// loadInstance loads a single persisted instance and adds it to the registry
func (im *instanceManager) loadInstance(persistedInst *instance.Instance) error {
	name := persistedInst.Name
	options := persistedInst.GetOptions()

	// Check if this is a remote instance (local node not in the Nodes set)
	var isRemote bool
	var nodeName string
	if options != nil {
		if _, isLocal := options.Nodes[im.globalConfig.LocalNode]; !isLocal && len(options.Nodes) > 0 {
			// Get the first node from the set
			for node := range options.Nodes {
				nodeName = node
				isRemote = true
				break
			}
		}
	}

	var statusCallback func(oldStatus, newStatus instance.Status)
	if !isRemote {
		// Only set status callback for local instances
		statusCallback = func(oldStatus, newStatus instance.Status) {
			im.onStatusChange(name, oldStatus, newStatus)
		}
	}

	// Create new inst using NewInstance (handles validation, defaults, setup)
	inst := instance.New(name, im.globalConfig, options, statusCallback)

	// Restore persisted fields that NewInstance doesn't set
	inst.Created = persistedInst.Created
	inst.SetStatus(persistedInst.GetStatus())

	// Handle remote instance mapping
	if isRemote {
		// Map instance to node in remote manager
		if err := im.remote.setInstanceNode(name, nodeName); err != nil {
			return fmt.Errorf("failed to set instance node: %w", err)
		}
	} else {
		// Allocate port for local instances
		if inst.GetPort() > 0 {
			port := inst.GetPort()
			if err := im.ports.allocateSpecific(port, name); err != nil {
				return fmt.Errorf("port conflict: instance %s wants port %d which is already in use: %w", name, port, err)
			}
		}
	}

	// Add instance to registry
	if err := im.registry.add(inst); err != nil {
		return fmt.Errorf("failed to add instance to registry: %w", err)
	}

	return nil
}

// autoStartInstances starts instances that were running when persisted and have auto-restart enabled
// For instances with auto-restart disabled, it sets their status to Stopped
func (im *instanceManager) autoStartInstances() {
	instances := im.registry.list()

	var instancesToStart []*instance.Instance
	var instancesToStop []*instance.Instance

	for _, inst := range instances {
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

	// Stop instances that have auto-restart disabled
	for _, inst := range instancesToStop {
		log.Printf("Instance %s was running but auto-restart is disabled, setting status to stopped", inst.Name)
		inst.SetStatus(instance.Stopped)
		im.registry.markStopped(inst.Name)
	}

	// Start instances that have auto-restart enabled
	for _, inst := range instancesToStart {
		log.Printf("Auto-starting instance %s", inst.Name)
		// Reset running state before starting (since Start() expects stopped instance)
		inst.SetStatus(instance.Stopped)
		im.registry.markStopped(inst.Name)

		// Check if this is a remote instance
		if node, exists := im.remote.getNodeForInstance(inst.Name); exists && node != nil {
			// Remote instance - use remote manager with context
			ctx := context.Background()
			if _, err := im.remote.startInstance(ctx, node, inst.Name); err != nil {
				log.Printf("Failed to auto-start remote instance %s: %v", inst.Name, err)
			}
		} else {
			// Local instance - call Start() directly
			if err := inst.Start(); err != nil {
				log.Printf("Failed to auto-start instance %s: %v", inst.Name, err)
			}
		}
	}
}

func (im *instanceManager) onStatusChange(name string, oldStatus, newStatus instance.Status) {
	if newStatus == instance.Running {
		im.registry.markRunning(name)
	} else {
		im.registry.markStopped(name)
	}
}

// getNodeForInstance returns the node configuration for a remote instance
// Returns nil if the instance is not remote or the node is not found
func (im *instanceManager) getNodeForInstance(inst *instance.Instance) *config.NodeConfig {
	if !inst.IsRemote() {
		return nil
	}

	// Check if we have a node mapping in remote manager
	if nodeConfig, exists := im.remote.getNodeForInstance(inst.Name); exists {
		return nodeConfig
	}

	return nil
}

// lockInstance returns the lock for a specific instance, creating one if needed.
// This allows concurrent operations on different instances while preventing
// concurrent operations on the same instance.
func (im *instanceManager) lockInstance(name string) *sync.Mutex {
	lock, _ := im.instanceLocks.LoadOrStore(name, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// unlockAndCleanup unlocks the instance lock and removes it from the map.
// This should only be called when deleting an instance to prevent memory leaks.
func (im *instanceManager) unlockAndCleanup(name string) {
	if lock, ok := im.instanceLocks.Load(name); ok {
		lock.(*sync.Mutex).Unlock()
		im.instanceLocks.Delete(name)
	}
}
