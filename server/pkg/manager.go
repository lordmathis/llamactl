package llamactl

import (
	"fmt"

	"github.com/google/uuid"
)

// InstanceManager defines the interface for managing instances of the llama server.
type InstanceManager interface {
	ListInstances() ([]*Instance, error)
	CreateInstance(options *InstanceOptions) (*Instance, error)
	GetInstance(id uuid.UUID) (*Instance, error)
	UpdateInstance(id uuid.UUID, options *InstanceOptions) (*Instance, error)
	DeleteInstance(id uuid.UUID) error
	StartInstance(id uuid.UUID) (*Instance, error)
	StopInstance(id uuid.UUID) (*Instance, error)
	RestartInstance(id uuid.UUID) (*Instance, error)
	GetInstanceLogs(id uuid.UUID) (string, error)
}

type instanceManager struct {
	instances map[uuid.UUID]*Instance
	portRange [][2]int // Range of ports to use for instances
	ports     map[int]bool
}

// NewInstanceManager creates a new instance of InstanceManager.
func NewInstanceManager() InstanceManager {
	return &instanceManager{
		instances: make(map[uuid.UUID]*Instance),
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

	// Generate a unique ID for the new instance
	id := uuid.New()
	for im.instances[id] != nil {
		id = uuid.New() // Ensure unique ID
	}

	// Assign a port if not specified
	if options.Port == 0 {
		port, err := im.getNextAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to get next available port: %w", err)
		}
		options.Port = port
	}

	instance := NewInstance(id, options)
	im.instances[instance.ID] = instance

	return instance, nil

}

// GetInstance retrieves an instance by its ID.
func (im *instanceManager) GetInstance(id uuid.UUID) (*Instance, error) {
	instance, exists := im.instances[id]
	if !exists {
		return nil, fmt.Errorf("instance with ID %s not found", id)
	}
	return instance, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
func (im *instanceManager) UpdateInstance(id uuid.UUID, options *InstanceOptions) (*Instance, error) {
	instance, exists := im.instances[id]
	if !exists {
		return nil, fmt.Errorf("instance with ID %s not found", id)
	}
	instance.Options = options
	return instance, nil
}

// DeleteInstance removes stopped instance by its ID.
func (im *instanceManager) DeleteInstance(id uuid.UUID) error {
	_, exists := im.instances[id]
	if !exists {
		return fmt.Errorf("instance with ID %s not found", id)
	}

	if im.instances[id].Running {
		return fmt.Errorf("instance with ID %s is still running, stop it before deleting", id)
	}

	delete(im.instances, id)
	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(id uuid.UUID) (*Instance, error) {
	instance, exists := im.instances[id]
	if !exists {
		return nil, fmt.Errorf("instance with ID %s not found", id)
	}
	if instance.Running {
		return instance, fmt.Errorf("instance with ID %s is already running", id)
	}

	//TODO:
	return instance, nil
}

// StopInstance stops a running instance and returns it.
func (im *instanceManager) StopInstance(id uuid.UUID) (*Instance, error) {
	instance, exists := im.instances[id]
	if !exists {
		return nil, fmt.Errorf("instance with ID %s not found", id)
	}
	if !instance.Running {
		return instance, fmt.Errorf("instance with ID %s is already stopped", id)
	}

	// TODO:
	return instance, nil
}

// RestartInstance stops and then starts an instance, returning the updated instance.
func (im *instanceManager) RestartInstance(id uuid.UUID) (*Instance, error) {
	instance, err := im.StopInstance(id)
	if err != nil {
		return nil, err
	}
	return im.StartInstance(instance.ID)
}

// GetInstanceLogs retrieves the logs for a specific instance by its ID.
func (im *instanceManager) GetInstanceLogs(id uuid.UUID) (string, error) {
	_, exists := im.instances[id]
	if !exists {
		return "", fmt.Errorf("instance with ID %s not found", id)
	}

	// TODO: Implement actual log retrieval logic
	return fmt.Sprintf("Logs for instance %s", id), nil
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
