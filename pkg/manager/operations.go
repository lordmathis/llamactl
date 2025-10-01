package manager

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"os"
	"path/filepath"
)

type MaxRunningInstancesError error

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

	// Assign and validate port for backend-specific options
	if err := im.assignAndValidatePort(options); err != nil {
		return nil, err
	}

	statusCallback := func(oldStatus, newStatus instance.InstanceStatus) {
		im.onStatusChange(name, oldStatus, newStatus)
	}

	inst := instance.NewInstance(name, &im.backendsConfig, &im.instancesConfig, options, statusCallback)
	im.instances[inst.Name] = inst

	if err := im.persistInstance(inst); err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return inst, nil
}

// GetInstance retrieves an instance by its name.
func (im *instanceManager) GetInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.GetRemoteInstance(node, name)
	}

	return inst, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
// If the instance is running, it will be restarted to apply the new options.
func (im *instanceManager) UpdateInstance(name string, options *instance.CreateInstanceOptions) (*instance.Process, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.UpdateRemoteInstance(node, name, options)
	}

	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	err := validation.ValidateInstanceOptions(options)
	if err != nil {
		return nil, err
	}

	// Check if instance is running before updating options
	wasRunning := inst.IsRunning()

	// If the instance is running, stop it first
	if wasRunning {
		if err := inst.Stop(); err != nil {
			return nil, fmt.Errorf("failed to stop instance %s for update: %w", name, err)
		}
	}

	// Now update the options while the instance is stopped
	inst.SetOptions(options)

	// If it was running before, start it again with the new options
	if wasRunning {
		if err := inst.Start(); err != nil {
			return nil, fmt.Errorf("failed to start instance %s after update: %w", name, err)
		}
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	if err := im.persistInstance(inst); err != nil {
		return nil, fmt.Errorf("failed to persist updated instance %s: %w", name, err)
	}

	return inst, nil
}

// DeleteInstance removes stopped instance by its name.
func (im *instanceManager) DeleteInstance(name string) error {
	im.mu.Lock()
	inst, exists := im.instances[name]
	im.mu.Unlock()

	if !exists {
		return fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.DeleteRemoteInstance(node, name)
	}

	if inst.IsRunning() {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.ports, inst.GetPort())
	delete(im.instances, name)

	// Delete the instance's config file if persistence is enabled
	instancePath := filepath.Join(im.instancesConfig.InstancesDir, inst.Name+".json")
	if err := os.Remove(instancePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file for instance %s: %w", inst.Name, err)
	}

	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	maxRunningExceeded := len(im.runningInstances) >= im.instancesConfig.MaxRunningInstances && im.instancesConfig.MaxRunningInstances != -1
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.StartRemoteInstance(node, name)
	}

	if inst.IsRunning() {
		return inst, fmt.Errorf("instance with name %s is already running", name)
	}

	if maxRunningExceeded {
		return nil, MaxRunningInstancesError(fmt.Errorf("maximum number of running instances (%d) reached", im.instancesConfig.MaxRunningInstances))
	}

	if err := inst.Start(); err != nil {
		return nil, fmt.Errorf("failed to start instance %s: %w", name, err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	err := im.persistInstance(inst)
	if err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return inst, nil
}

func (im *instanceManager) IsMaxRunningInstancesReached() bool {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if im.instancesConfig.MaxRunningInstances != -1 && len(im.runningInstances) >= im.instancesConfig.MaxRunningInstances {
		return true
	}

	return false
}

// StopInstance stops a running instance and returns it.
func (im *instanceManager) StopInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.StopRemoteInstance(node, name)
	}

	if !inst.IsRunning() {
		return inst, fmt.Errorf("instance with name %s is already stopped", name)
	}

	if err := inst.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop instance %s: %w", name, err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	err := im.persistInstance(inst)
	if err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return inst, nil
}

// RestartInstance stops and then starts an instance, returning the updated instance.
func (im *instanceManager) RestartInstance(name string) (*instance.Process, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.RestartRemoteInstance(node, name)
	}

	inst, err := im.StopInstance(name)
	if err != nil {
		return nil, err
	}
	return im.StartInstance(inst.Name)
}

// GetInstanceLogs retrieves the logs for a specific instance by its name.
func (im *instanceManager) GetInstanceLogs(name string) (string, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.GetRemoteInstanceLogs(node, name)
	}

	// TODO: Implement actual log retrieval logic
	return fmt.Sprintf("Logs for instance %s", name), nil
}

// getPortFromOptions extracts the port from backend-specific options
func (im *instanceManager) getPortFromOptions(options *instance.CreateInstanceOptions) int {
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		if options.LlamaServerOptions != nil {
			return options.LlamaServerOptions.Port
		}
	case backends.BackendTypeMlxLm:
		if options.MlxServerOptions != nil {
			return options.MlxServerOptions.Port
		}
	case backends.BackendTypeVllm:
		if options.VllmServerOptions != nil {
			return options.VllmServerOptions.Port
		}
	}
	return 0
}

// setPortInOptions sets the port in backend-specific options
func (im *instanceManager) setPortInOptions(options *instance.CreateInstanceOptions, port int) {
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		if options.LlamaServerOptions != nil {
			options.LlamaServerOptions.Port = port
		}
	case backends.BackendTypeMlxLm:
		if options.MlxServerOptions != nil {
			options.MlxServerOptions.Port = port
		}
	case backends.BackendTypeVllm:
		if options.VllmServerOptions != nil {
			options.VllmServerOptions.Port = port
		}
	}
}

// assignAndValidatePort assigns a port if not specified and validates it's not in use
func (im *instanceManager) assignAndValidatePort(options *instance.CreateInstanceOptions) error {
	currentPort := im.getPortFromOptions(options)

	if currentPort == 0 {
		// Assign a port if not specified
		port, err := im.getNextAvailablePort()
		if err != nil {
			return fmt.Errorf("failed to get next available port: %w", err)
		}
		im.setPortInOptions(options, port)
		// Mark the port as used
		im.ports[port] = true
	} else {
		// Validate the specified port
		if _, exists := im.ports[currentPort]; exists {
			return fmt.Errorf("port %d is already in use", currentPort)
		}
		// Mark the port as used
		im.ports[currentPort] = true
	}

	return nil
}
