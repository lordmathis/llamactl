package manager

import (
	"fmt"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"os"
	"path/filepath"
)

// ListInstances returns a list of all instances managed by the instance manager.
func (im *instanceManager) ListInstances() ([]*instance.Process, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	instances := make([]*instance.Process, 0, len(im.instances))
	for _, inst := range im.instances {
		instances = append(instances, inst)
	}
	return instances, nil
}

// CreateInstance creates a new instance with the given options and returns it.
// The instance is initially in a "stopped" state.
func (im *instanceManager) CreateInstance(name string, options *instance.CreateInstanceOptions) (*instance.Process, error) {
	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	name, err := validation.ValidateInstanceName(name)
	if err != nil {
		return nil, err
	}

	err = validation.ValidateInstanceOptions(options)
	if err != nil {
		return nil, err
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	// Check max instances limit after acquiring the lock
	if len(im.instances) >= im.instancesConfig.MaxInstances && im.instancesConfig.MaxInstances != -1 {
		return nil, fmt.Errorf("maximum number of instances (%d) reached", im.instancesConfig.MaxInstances)
	}

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

	inst := instance.NewInstance(name, &im.instancesConfig, options)
	im.instances[inst.Name] = inst
	im.ports[options.Port] = true

	if err := im.persistInstance(inst); err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return inst, nil
}

// GetInstance retrieves an instance by its name.
func (im *instanceManager) GetInstance(name string) (*instance.Process, error) {
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
func (im *instanceManager) UpdateInstance(name string, options *instance.CreateInstanceOptions) (*instance.Process, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	err := validation.ValidateInstanceOptions(options)
	if err != nil {
		return nil, err
	}

	// Check if instance is running before updating options
	wasRunning := instance.IsRunning()

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

	im.mu.Lock()
	defer im.mu.Unlock()
	if err := im.persistInstance(instance); err != nil {
		return nil, fmt.Errorf("failed to persist updated instance %s: %w", name, err)
	}

	return instance, nil
}

// DeleteInstance removes stopped instance by its name.
func (im *instanceManager) DeleteInstance(name string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	instance, exists := im.instances[name]
	if !exists {
		return fmt.Errorf("instance with name %s not found", name)
	}

	if instance.IsRunning() {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	delete(im.ports, instance.GetOptions().Port)
	delete(im.instances, name)

	// Delete the instance's config file if persistence is enabled
	instancePath := filepath.Join(im.instancesConfig.InstancesDir, instance.Name+".json")
	if err := os.Remove(instancePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file for instance %s: %w", instance.Name, err)
	}

	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	if instance.IsRunning() {
		return instance, fmt.Errorf("instance with name %s is already running", name)
	}

	if err := instance.Start(); err != nil {
		return nil, fmt.Errorf("failed to start instance %s: %w", name, err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	err := im.persistInstance(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return instance, nil
}

// StopInstance stops a running instance and returns it.
func (im *instanceManager) StopInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	instance, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}
	if !instance.IsRunning() {
		return instance, fmt.Errorf("instance with name %s is already stopped", name)
	}

	if err := instance.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop instance %s: %w", name, err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	err := im.persistInstance(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return instance, nil
}

// RestartInstance stops and then starts an instance, returning the updated instance.
func (im *instanceManager) RestartInstance(name string) (*instance.Process, error) {
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
