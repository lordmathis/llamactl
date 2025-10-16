package manager

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"os"
	"path/filepath"
)

type MaxRunningInstancesError error

// updateLocalInstanceFromRemote updates the local stub instance with data from the remote instance
// while preserving the Nodes field to maintain remote instance tracking
func (im *instanceManager) updateLocalInstanceFromRemote(localInst *instance.Instance, remoteInst *instance.Instance) {
	if localInst == nil || remoteInst == nil {
		return
	}

	// Get the remote instance options
	remoteOptions := remoteInst.GetOptions()
	if remoteOptions == nil {
		return
	}

	// Preserve the Nodes field from the local instance
	localOptions := localInst.GetOptions()
	var preservedNodes []string
	if localOptions != nil && len(localOptions.Nodes) > 0 {
		preservedNodes = make([]string, len(localOptions.Nodes))
		copy(preservedNodes, localOptions.Nodes)
	}

	// Create a copy of remote options and restore the Nodes field
	updatedOptions := *remoteOptions
	updatedOptions.Nodes = preservedNodes

	// Update the local instance with all remote data
	localInst.SetOptions(&updatedOptions)
	localInst.SetStatus(remoteInst.GetStatus())
	localInst.Created = remoteInst.Created
}

// ListInstances returns a list of all instances managed by the instance manager.
// For remote instances, this fetches the live state from remote nodes and updates local stubs.
func (im *instanceManager) ListInstances() ([]*instance.Instance, error) {
	im.mu.RLock()
	localInstances := make([]*instance.Instance, 0, len(im.instances))
	for _, inst := range im.instances {
		localInstances = append(localInstances, inst)
	}
	im.mu.RUnlock()

	// Update remote instances with live state
	for _, inst := range localInstances {
		if node := im.getNodeForInstance(inst); node != nil {
			remoteInst, err := im.GetRemoteInstance(node, inst.Name)
			if err != nil {
				// Log error but continue with stale data
				// Don't fail the entire list operation due to one remote failure
				continue
			}

			// Update the local stub with all remote data (preserving Nodes)
			im.mu.Lock()
			im.updateLocalInstanceFromRemote(inst, remoteInst)
			im.mu.Unlock()
		}
	}

	return localInstances, nil
}

// CreateInstance creates a new instance with the given options and returns it.
// The instance is initially in a "stopped" state.
func (im *instanceManager) CreateInstance(name string, options *instance.Options) (*instance.Instance, error) {
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

	// Check if instance with this name already exists (must be globally unique)
	if im.instances[name] != nil {
		return nil, fmt.Errorf("instance with name %s already exists", name)
	}

	// Check if this is a remote instance
	// An instance is remote if Nodes is specified AND the first node is not the local node
	isRemote := len(options.Nodes) > 0 && options.Nodes[0] != im.localNodeName
	var nodeConfig *config.NodeConfig

	if isRemote {
		// Validate that the node exists
		nodeName := options.Nodes[0] // Use first node for now
		var exists bool
		nodeConfig, exists = im.nodeConfigMap[nodeName]
		if !exists {
			return nil, fmt.Errorf("node %s not found", nodeName)
		}

		// Create the remote instance on the remote node
		remoteInst, err := im.CreateRemoteInstance(nodeConfig, name, options)
		if err != nil {
			return nil, err
		}

		// Create a local stub that preserves the Nodes field for tracking
		// We keep the original options (with Nodes) so IsRemote() works correctly
		inst := instance.NewInstance(name, &im.backendsConfig, &im.instancesConfig, options, im.localNodeName, nil)

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		// Add to local tracking maps (but don't count towards limits)
		im.instances[name] = inst
		im.instanceNodeMap[name] = nodeConfig

		// Persist the remote instance locally for tracking across restarts
		if err := im.persistInstance(inst); err != nil {
			return nil, fmt.Errorf("failed to persist remote instance %s: %w", name, err)
		}

		return inst, nil
	}

	// Local instance creation
	// Check max instances limit for local instances only
	localInstanceCount := len(im.instances) - len(im.instanceNodeMap)
	if localInstanceCount >= im.instancesConfig.MaxInstances && im.instancesConfig.MaxInstances != -1 {
		return nil, fmt.Errorf("maximum number of instances (%d) reached", im.instancesConfig.MaxInstances)
	}

	// Assign and validate port for backend-specific options
	if err := im.assignAndValidatePort(options); err != nil {
		return nil, err
	}

	statusCallback := func(oldStatus, newStatus instance.Status) {
		im.onStatusChange(name, oldStatus, newStatus)
	}

	inst := instance.NewInstance(name, &im.backendsConfig, &im.instancesConfig, options, im.localNodeName, statusCallback)
	im.instances[inst.Name] = inst

	if err := im.persistInstance(inst); err != nil {
		return nil, fmt.Errorf("failed to persist instance %s: %w", name, err)
	}

	return inst, nil
}

// GetInstance retrieves an instance by its name.
// For remote instances, this fetches the live state from the remote node and updates the local stub.
func (im *instanceManager) GetInstance(name string) (*instance.Instance, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and fetch live state
	if node := im.getNodeForInstance(inst); node != nil {
		remoteInst, err := im.GetRemoteInstance(node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.mu.Lock()
		im.updateLocalInstanceFromRemote(inst, remoteInst)
		im.mu.Unlock()

		// Return the local stub (preserving Nodes field)
		return inst, nil
	}

	return inst, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
// If the instance is running, it will be restarted to apply the new options.
func (im *instanceManager) UpdateInstance(name string, options *instance.Options) (*instance.Instance, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		remoteInst, err := im.UpdateRemoteInstance(node, name, options)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.mu.Lock()
		im.updateLocalInstanceFromRemote(inst, remoteInst)
		im.mu.Unlock()

		// Persist the updated remote instance locally
		im.mu.Lock()
		defer im.mu.Unlock()
		if err := im.persistInstance(inst); err != nil {
			return nil, fmt.Errorf("failed to persist updated remote instance %s: %w", name, err)
		}

		return inst, nil
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
		err := im.DeleteRemoteInstance(node, name)
		if err != nil {
			return err
		}

		// Clean up local tracking
		im.mu.Lock()
		defer im.mu.Unlock()
		delete(im.instances, name)
		delete(im.instanceNodeMap, name)

		// Delete the instance's config file if persistence is enabled
		// Re-validate instance name for security (defense in depth)
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			return fmt.Errorf("invalid instance name for file deletion: %w", err)
		}
		instancePath := filepath.Join(im.instancesConfig.InstancesDir, validatedName+".json")
		if err := os.Remove(instancePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete config file for remote instance %s: %w", validatedName, err)
		}

		return nil
	}

	if inst.IsRunning() {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.ports, inst.GetPort())
	delete(im.instances, name)

	// Delete the instance's config file if persistence is enabled
	// Re-validate instance name for security (defense in depth)
	validatedName, err := validation.ValidateInstanceName(inst.Name)
	if err != nil {
		return fmt.Errorf("invalid instance name for file deletion: %w", err)
	}
	instancePath := filepath.Join(im.instancesConfig.InstancesDir, validatedName+".json")
	if err := os.Remove(instancePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file for instance %s: %w", validatedName, err)
	}

	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*instance.Instance, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		remoteInst, err := im.StartRemoteInstance(node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.mu.Lock()
		im.updateLocalInstanceFromRemote(inst, remoteInst)
		im.mu.Unlock()

		return inst, nil
	}

	if inst.IsRunning() {
		return inst, fmt.Errorf("instance with name %s is already running", name)
	}

	// Check max running instances limit for local instances only
	im.mu.RLock()
	localRunningCount := 0
	for instName := range im.runningInstances {
		if _, isRemote := im.instanceNodeMap[instName]; !isRemote {
			localRunningCount++
		}
	}
	maxRunningExceeded := localRunningCount >= im.instancesConfig.MaxRunningInstances && im.instancesConfig.MaxRunningInstances != -1
	im.mu.RUnlock()

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
func (im *instanceManager) StopInstance(name string) (*instance.Instance, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		remoteInst, err := im.StopRemoteInstance(node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.mu.Lock()
		im.updateLocalInstanceFromRemote(inst, remoteInst)
		im.mu.Unlock()

		return inst, nil
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
func (im *instanceManager) RestartInstance(name string) (*instance.Instance, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		remoteInst, err := im.RestartRemoteInstance(node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.mu.Lock()
		im.updateLocalInstanceFromRemote(inst, remoteInst)
		im.mu.Unlock()

		return inst, nil
	}

	inst, err := im.StopInstance(name)
	if err != nil {
		return nil, err
	}
	return im.StartInstance(inst.Name)
}

// GetInstanceLogs retrieves the logs for a specific instance by its name.
func (im *instanceManager) GetInstanceLogs(name string, numLines int) (string, error) {
	im.mu.RLock()
	inst, exists := im.instances[name]
	im.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		return im.GetRemoteInstanceLogs(node, name, numLines)
	}

	// Get logs from the local instance
	return inst.GetLogs(numLines)
}

// getPortFromOptions extracts the port from backend-specific options
func (im *instanceManager) getPortFromOptions(options *instance.Options) int {
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
func (im *instanceManager) setPortInOptions(options *instance.Options, port int) {
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
func (im *instanceManager) assignAndValidatePort(options *instance.Options) error {
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
