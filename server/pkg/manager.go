package llamactl

import (
	"fmt"
)

// InstanceManager defines the interface for managing instances of the llama server.
type InstanceManager interface {
	ListInstances() ([]*Instance, error)
	CreateInstance(options *InstanceOptions) (*Instance, error)
	GetInstance(name string) (*Instance, error)
	UpdateInstance(name string, options *InstanceOptions) (*Instance, error)
	DeleteInstance(name string) error
	StartInstance(name string) (*Instance, error)
	StopInstance(name string) (*Instance, error)
	RestartInstance(name string) (*Instance, error)
	GetInstanceLogs(name string) (string, error)
}

type instanceManager struct {
	instances map[string]*Instance
	portRange [][2]int // Range of ports to use for instances
	ports     map[int]bool
}

// NewInstanceManager creates a new instance of InstanceManager.
func NewInstanceManager() InstanceManager {
	return &instanceManager{
		instances: make(map[string]*Instance),
		portRange: [][2]int{{8000, 9000}},
		ports:     make(map[int]bool),
	}
}

// ListInstances returns a list of all instances managed by the instance manager.
func (im *instanceManager) ListInstances() ([]*Instance, error) {
	var instances []*Instance
	for _, instance := range im.instances {
		instances = append(instances, instance)
	}
	return instances, nil
}

// CreateInstance creates a new instance with the given options and returns it.
// The instance is initially in a "stopped" state.
func (im *instanceManager) CreateInstance(options *InstanceOptions) (*Instance, error) {
	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	// Check if name is provided
	if options.Name == "" {
		return nil, fmt.Errorf("instance name cannot be empty")
	}

	// Check if instance with this name already exists
	if im.instances[options.Name] != nil {
		return nil, fmt.Errorf("instance with name %s already exists", options.Name)
	}

	// Assign a port if not specified
	if options.Port == 0 {
		port, err := im.getNextAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to get next available port: %w", err)
		}
		options.Port = port
	}

	instance := NewInstance(options.Name, options)
	im.instances[instance.Name] = instance

	return instance, nil

}

// GetInstance retrieves an instance by its name.
func (im *instanceManager) GetInstance(name string) (*Instance, error) {
	instance, exists := im.instances[name]
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	return instance, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
func (im *instanceManager) UpdateInstance(name string, options *InstanceOptions) (*Instance, error) {
	instance, exists := im.instances[name]
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	instance.SetOptions(options)
	return instance, nil
}

// DeleteInstance removes stopped instance by its name.
func (im *instanceManager) DeleteInstance(name string) error {
	_, exists := im.instances[name]
	if !exists {
		return fmt.Errorf("instance with name %s not found", name)
	}

	if im.instances[name].Running {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	delete(im.instances, name)
	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*Instance, error) {
	instance, exists := im.instances[name]
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
	instance, exists := im.instances[name]
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
	_, exists := im.instances[name]
	if !exists {
		return "", fmt.Errorf("instance with name %s not found", name)
	}

	// TODO: Implement actual log retrieval logic
	return fmt.Sprintf("Logs for instance %s", name), nil
}

func (im *instanceManager) getNextAvailablePort() (int, error) {
	for _, portRange := range im.portRange {
		for port := portRange[0]; port <= portRange[1]; port++ {
			if !im.ports[port] {
				im.ports[port] = true
				return port, nil
			}
		}
	}
	return 0, fmt.Errorf("no available ports in the specified range")
}
