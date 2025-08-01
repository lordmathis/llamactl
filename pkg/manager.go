package llamactl

import (
	"fmt"
	"sync"
)

// InstanceManager defines the interface for managing instances of the llama server.
type InstanceManager interface {
	ListInstances() ([]*Instance, error)
	CreateInstance(name string, options *CreateInstanceOptions) (*Instance, error)
	GetInstance(name string) (*Instance, error)
	UpdateInstance(name string, options *CreateInstanceOptions) (*Instance, error)
	DeleteInstance(name string) error
	StartInstance(name string) (*Instance, error)
	StopInstance(name string) (*Instance, error)
	RestartInstance(name string) (*Instance, error)
	GetInstanceLogs(name string) (string, error)
	Shutdown()
}

type instanceManager struct {
	mu              sync.RWMutex
	instances       map[string]*Instance
	ports           map[int]bool
	instancesConfig InstancesConfig
}

// NewInstanceManager creates a new instance of InstanceManager.
func NewInstanceManager(instancesConfig InstancesConfig) InstanceManager {
	return &instanceManager{
		instances:       make(map[string]*Instance),
		ports:           make(map[int]bool),
		instancesConfig: instancesConfig,
	}
}

// ListInstances returns a list of all instances managed by the instance manager.
func (im *instanceManager) ListInstances() ([]*Instance, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	instances := make([]*Instance, 0, len(im.instances))
	for _, instance := range im.instances {
		instances = append(instances, instance)
	}
	return instances, nil
}

// CreateInstance creates a new instance with the given options and returns it.
// The instance is initially in a "stopped" state.
func (im *instanceManager) CreateInstance(name string, options *CreateInstanceOptions) (*Instance, error) {
	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	if len(im.instances) >= im.instancesConfig.MaxInstances && im.instancesConfig.MaxInstances != -1 {
		return nil, fmt.Errorf("maximum number of instances (%d) reached", im.instancesConfig.MaxInstances)
	}

	err := ValidateInstanceName(name)
	if err != nil {
		return nil, err
	}

	err = ValidateInstanceOptions(options)
	if err != nil {
		return nil, err
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	// Check if instance with this name already exists
	if im.instances[name] != nil {
		return nil, fmt.Errorf("instance with name %s already exists", name)
	}

	// Assign a port if not specified
	if options.Port == 0 {
		port, err := im.getNextAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to get next available port: %w", err)
		}
		options.Port = port
	} else {
		// Validate the specified port
		if _, exists := im.ports[options.Port]; exists {
			return nil, fmt.Errorf("port %d is already in use", options.Port)
		}
		im.ports[options.Port] = true
	}

	instance := NewInstance(name, &im.instancesConfig, options)
	im.instances[instance.Name] = instance
	im.ports[options.Port] = true

	return instance, nil
}

// GetInstance retrieves an instance by its name.
func (im *instanceManager) GetInstance(name string) (*Instance, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	instance, exists := im.instances[name]
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	return instance, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
// If the instance is running, it will be restarted to apply the new options.
func (im *instanceManager) UpdateInstance(name string, options *CreateInstanceOptions) (*Instance, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	err := ValidateInstanceOptions(options)
	if err != nil {
		return nil, err
	}

	// Check if instance is running before updating options
	wasRunning := instance.Running

	// If the instance is running, stop it first
	if wasRunning {
		if err := instance.Stop(); err != nil {
			return nil, fmt.Errorf("failed to stop instance %s for update: %w", name, err)
		}
	}

	// Now update the options while the instance is stopped
	instance.SetOptions(options)

	// If it was running before, start it again with the new options
	if wasRunning {
		if err := instance.Start(); err != nil {
			return nil, fmt.Errorf("failed to start instance %s after update: %w", name, err)
		}
	}

	return instance, nil
}

// DeleteInstance removes stopped instance by its name.
func (im *instanceManager) DeleteInstance(name string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	_, exists := im.instances[name]
	if !exists {
		return fmt.Errorf("instance with name %s not found", name)
	}

	if im.instances[name].Running {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	delete(im.ports, im.instances[name].options.Port)
	delete(im.instances, name)
	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*Instance, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	if instance.Running {
		return instance, fmt.Errorf("instance with name %s is already running", name)
	}

	if err := instance.Start(); err != nil {
		return nil, fmt.Errorf("failed to start instance %s: %w", name, err)
	}

	return instance, nil
}

// StopInstance stops a running instance and returns it.
func (im *instanceManager) StopInstance(name string) (*Instance, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	if !instance.Running {
		return instance, fmt.Errorf("instance with name %s is already stopped", name)
	}

	if err := instance.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop instance %s: %w", name, err)
	}

	return instance, nil
}

// RestartInstance stops and then starts an instance, returning the updated instance.
func (im *instanceManager) RestartInstance(name string) (*Instance, error) {
	instance, err := im.StopInstance(name)
	if err != nil {
		return nil, err
	}
	return im.StartInstance(instance.Name)
}

// GetInstanceLogs retrieves the logs for a specific instance by its name.
func (im *instanceManager) GetInstanceLogs(name string) (string, error) {
	im.mu.RLock()
	_, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("instance with name %s not found", name)
	}

	// TODO: Implement actual log retrieval logic
	return fmt.Sprintf("Logs for instance %s", name), nil
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

func (im *instanceManager) Shutdown() {
	im.mu.Lock()
	defer im.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(im.instances))

	for name, instance := range im.instances {
		if !instance.Running {
			wg.Done() // If instance is not running, just mark it as done
			continue
		}

		go func(name string, instance *Instance) {
			defer wg.Done()
			fmt.Printf("Stopping instance %s...\n", name)
			// Attempt to stop the instance gracefully
			if err := instance.Stop(); err != nil {
				fmt.Printf("Error stopping instance %s: %v\n", name, err)
			}
		}(name, instance)
	}

	wg.Wait()
	fmt.Println("All instances stopped.")
}
