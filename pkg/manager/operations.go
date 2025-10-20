package manager

import (
	"context"
	"fmt"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"log"
)

type MaxRunningInstancesError error

// updateLocalInstanceFromRemote updates the local stub instance with data from the remote instance
func (im *instanceManager) updateLocalInstanceFromRemote(localInst *instance.Instance, remoteInst *instance.Instance) {
	if localInst == nil || remoteInst == nil {
		return
	}

	remoteOptions := remoteInst.GetOptions()
	if remoteOptions == nil {
		return
	}

	// Update the local instance with all remote data
	localInst.SetOptions(remoteOptions)
	localInst.SetStatus(remoteInst.GetStatus())
	localInst.Created = remoteInst.Created
}

// ListInstances returns a list of all instances managed by the instance manager.
// For remote instances, this fetches the live state from remote nodes and updates local stubs.
func (im *instanceManager) ListInstances() ([]*instance.Instance, error) {
	instances := im.registry.List()

	// Update remote instances with live state
	ctx := context.Background()
	for _, inst := range instances {
		if node := im.getNodeForInstance(inst); node != nil {
			remoteInst, err := im.remote.GetInstance(ctx, node, inst.Name)
			if err != nil {
				// Log error but continue with stale data
				// Don't fail the entire list operation due to one remote failure
				continue
			}

			// Update the local stub with all remote data (preserving Nodes)
			im.updateLocalInstanceFromRemote(inst, remoteInst)
		}
	}

	return instances, nil
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

	err = options.BackendOptions.ValidateInstanceOptions()
	if err != nil {
		return nil, err
	}

	// Check if instance with this name already exists (must be globally unique)
	if _, exists := im.registry.Get(name); exists {
		return nil, fmt.Errorf("instance with name %s already exists", name)
	}

	// Check if this is a remote instance (local node not in the Nodes set)
	if _, isLocal := options.Nodes[im.localNodeName]; !isLocal && len(options.Nodes) > 0 {
		// Get the first node from the set
		var nodeName string
		for node := range options.Nodes {
			nodeName = node
			break
		}

		// Create the remote instance on the remote node
		ctx := context.Background()
		nodeConfig, exists := im.remote.GetNodeForInstance(nodeName)
		if !exists {
			// Try to set the node if it doesn't exist yet
			if err := im.remote.SetInstanceNode(name, nodeName); err != nil {
				return nil, fmt.Errorf("node %s not found", nodeName)
			}
			nodeConfig, _ = im.remote.GetNodeForInstance(name)
		}

		remoteInst, err := im.remote.CreateInstance(ctx, nodeConfig, name, options)
		if err != nil {
			return nil, err
		}

		// Create a local stub that preserves the Nodes field for tracking
		// We keep the original options (with Nodes) so IsRemote() works correctly
		inst := instance.New(name, &im.backendsConfig, &im.instancesConfig, options, im.localNodeName, nil)

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		// Map instance to node
		if err := im.remote.SetInstanceNode(name, nodeName); err != nil {
			return nil, fmt.Errorf("failed to map instance to node: %w", err)
		}

		// Add to registry (doesn't count towards local limits)
		if err := im.registry.Add(inst); err != nil {
			return nil, fmt.Errorf("failed to add instance to registry: %w", err)
		}

		// Persist the remote instance locally for tracking across restarts
		if err := im.persistInstance(inst); err != nil {
			// Rollback: remove from registry
			im.registry.Remove(name)
			return nil, fmt.Errorf("failed to persist remote instance %s: %w", name, err)
		}

		return inst, nil
	}

	// Local instance creation
	// Check max instances limit for local instances only
	totalInstances := im.registry.Count()
	remoteCount := 0
	for _, inst := range im.registry.List() {
		if inst.IsRemote() {
			remoteCount++
		}
	}
	localInstanceCount := totalInstances - remoteCount
	if localInstanceCount >= im.instancesConfig.MaxInstances && im.instancesConfig.MaxInstances != -1 {
		return nil, fmt.Errorf("maximum number of instances (%d) reached", im.instancesConfig.MaxInstances)
	}

	// Assign and validate port for backend-specific options
	currentPort := im.getPortFromOptions(options)
	var allocatedPort int
	if currentPort == 0 {
		// Allocate a port if not specified
		allocatedPort, err = im.ports.Allocate(name)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %w", err)
		}
		im.setPortInOptions(options, allocatedPort)
	} else {
		// Use the specified port
		if err := im.ports.AllocateSpecific(currentPort, name); err != nil {
			return nil, fmt.Errorf("port %d is already in use: %w", currentPort, err)
		}
		allocatedPort = currentPort
	}

	statusCallback := func(oldStatus, newStatus instance.Status) {
		im.onStatusChange(name, oldStatus, newStatus)
	}

	inst := instance.New(name, &im.backendsConfig, &im.instancesConfig, options, im.localNodeName, statusCallback)

	// Add to registry
	if err := im.registry.Add(inst); err != nil {
		// Rollback: release port
		im.ports.Release(allocatedPort)
		return nil, fmt.Errorf("failed to add instance to registry: %w", err)
	}

	// Persist instance (best-effort, don't fail if persistence fails)
	if err := im.persistInstance(inst); err != nil {
		log.Printf("Warning: failed to persist instance %s: %v", name, err)
	}

	return inst, nil
}

// GetInstance retrieves an instance by its name.
// For remote instances, this fetches the live state from the remote node and updates the local stub.
func (im *instanceManager) GetInstance(name string) (*instance.Instance, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and fetch live state
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		remoteInst, err := im.remote.GetInstance(ctx, node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		// Return the local stub (preserving Nodes field)
		return inst, nil
	}

	return inst, nil
}

// UpdateInstance updates the options of an existing instance and returns it.
// If the instance is running, it will be restarted to apply the new options.
func (im *instanceManager) UpdateInstance(name string, options *instance.Options) (*instance.Instance, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		remoteInst, err := im.remote.UpdateInstance(ctx, node, name, options)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		// Persist the updated remote instance locally
		if err := im.persistInstance(inst); err != nil {
			return nil, fmt.Errorf("failed to persist updated remote instance %s: %w", name, err)
		}

		return inst, nil
	}

	if options == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	err := options.BackendOptions.ValidateInstanceOptions()
	if err != nil {
		return nil, err
	}

	// Lock for local instance operations to prevent races
	im.operationMu.Lock()
	defer im.operationMu.Unlock()

	// Handle port changes
	oldPort := inst.GetPort()
	newPort := im.getPortFromOptions(options)
	var allocatedPort int

	if newPort != oldPort {
		// Port is changing - need to release old and allocate new
		if newPort == 0 {
			// Auto-allocate new port
			allocatedPort, err = im.ports.Allocate(name)
			if err != nil {
				return nil, fmt.Errorf("failed to allocate new port: %w", err)
			}
			im.setPortInOptions(options, allocatedPort)
		} else {
			// Use specified port
			if err := im.ports.AllocateSpecific(newPort, name); err != nil {
				return nil, fmt.Errorf("failed to allocate port %d: %w", newPort, err)
			}
			allocatedPort = newPort
		}

		// Release old port
		if oldPort > 0 {
			if err := im.ports.Release(oldPort); err != nil {
				// Rollback new port allocation
				im.ports.Release(allocatedPort)
				return nil, fmt.Errorf("failed to release old port %d: %w", oldPort, err)
			}
		}
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

	if err := im.persistInstance(inst); err != nil {
		return nil, fmt.Errorf("failed to persist updated instance %s: %w", name, err)
	}

	return inst, nil
}

// DeleteInstance removes stopped instance by its name.
func (im *instanceManager) DeleteInstance(name string) error {
	inst, exists := im.registry.Get(name)
	if !exists {
		return fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		err := im.remote.DeleteInstance(ctx, node, name)
		if err != nil {
			return err
		}

		// Clean up local tracking
		im.remote.RemoveInstance(name)
		im.registry.Remove(name)

		// Delete the instance's persistence file
		if err := im.persistence.Delete(name); err != nil {
			return fmt.Errorf("failed to delete config file for remote instance %s: %w", name, err)
		}

		return nil
	}

	// Lock for local instance operations to prevent races
	im.operationMu.Lock()
	defer im.operationMu.Unlock()

	if inst.IsRunning() {
		return fmt.Errorf("instance with name %s is still running, stop it before deleting", name)
	}

	// Release port (use ReleaseByInstance for proper cleanup)
	im.ports.ReleaseByInstance(name)

	// Remove from registry
	if err := im.registry.Remove(name); err != nil {
		return fmt.Errorf("failed to remove instance from registry: %w", err)
	}

	// Delete persistence file
	if err := im.persistence.Delete(name); err != nil {
		return fmt.Errorf("failed to delete config file for instance %s: %w", name, err)
	}

	return nil
}

// StartInstance starts a stopped instance and returns it.
// If the instance is already running, it returns an error.
func (im *instanceManager) StartInstance(name string) (*instance.Instance, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		remoteInst, err := im.remote.StartInstance(ctx, node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		return inst, nil
	}

	// Lock for local instance operations to prevent races
	im.operationMu.Lock()
	defer im.operationMu.Unlock()

	if inst.IsRunning() {
		return inst, fmt.Errorf("instance with name %s is already running", name)
	}

	// Check max running instances limit for local instances only
	if im.IsMaxRunningInstancesReached() {
		return nil, MaxRunningInstancesError(fmt.Errorf("maximum number of running instances (%d) reached", im.instancesConfig.MaxRunningInstances))
	}

	if err := inst.Start(); err != nil {
		return nil, fmt.Errorf("failed to start instance %s: %w", name, err)
	}

	// Persist instance (best-effort, don't fail if persistence fails)
	if err := im.persistInstance(inst); err != nil {
		log.Printf("Warning: failed to persist instance %s: %v", name, err)
	}

	return inst, nil
}

func (im *instanceManager) IsMaxRunningInstancesReached() bool {
	if im.instancesConfig.MaxRunningInstances == -1 {
		return false
	}

	// Count only local running instances (each node has its own limits)
	localRunningCount := 0
	for _, inst := range im.registry.ListRunning() {
		if !inst.IsRemote() {
			localRunningCount++
		}
	}

	return localRunningCount >= im.instancesConfig.MaxRunningInstances
}

// StopInstance stops a running instance and returns it.
func (im *instanceManager) StopInstance(name string) (*instance.Instance, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		remoteInst, err := im.remote.StopInstance(ctx, node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		return inst, nil
	}

	// Lock for local instance operations to prevent races
	im.operationMu.Lock()
	defer im.operationMu.Unlock()

	if !inst.IsRunning() {
		return inst, fmt.Errorf("instance with name %s is already stopped", name)
	}

	if err := inst.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop instance %s: %w", name, err)
	}

	// Persist instance (best-effort, don't fail if persistence fails)
	if err := im.persistInstance(inst); err != nil {
		log.Printf("Warning: failed to persist instance %s: %v", name, err)
	}

	return inst, nil
}

// RestartInstance stops and then starts an instance, returning the updated instance.
func (im *instanceManager) RestartInstance(name string) (*instance.Instance, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		remoteInst, err := im.remote.RestartInstance(ctx, node, name)
		if err != nil {
			return nil, err
		}

		// Update the local stub with all remote data (preserving Nodes)
		im.updateLocalInstanceFromRemote(inst, remoteInst)

		return inst, nil
	}

	// Lock for the entire restart operation to ensure atomicity
	im.operationMu.Lock()
	defer im.operationMu.Unlock()

	// Stop the instance
	if inst.IsRunning() {
		if err := inst.Stop(); err != nil {
			return nil, fmt.Errorf("failed to stop instance %s: %w", name, err)
		}
	}

	// Start the instance
	if err := inst.Start(); err != nil {
		return nil, fmt.Errorf("failed to start instance %s: %w", name, err)
	}

	// Persist the restarted instance
	if err := im.persistInstance(inst); err != nil {
		log.Printf("Warning: failed to persist instance %s: %v", name, err)
	}

	return inst, nil
}

// GetInstanceLogs retrieves the logs for a specific instance by its name.
func (im *instanceManager) GetInstanceLogs(name string, numLines int) (string, error) {
	inst, exists := im.registry.Get(name)
	if !exists {
		return "", fmt.Errorf("instance with name %s not found", name)
	}

	// Check if instance is remote and delegate to remote operation
	if node := im.getNodeForInstance(inst); node != nil {
		ctx := context.Background()
		return im.remote.GetInstanceLogs(ctx, node, name, numLines)
	}

	// Get logs from the local instance
	return inst.GetLogs(numLines)
}

// getPortFromOptions extracts the port from backend-specific options
func (im *instanceManager) getPortFromOptions(options *instance.Options) int {
	return options.BackendOptions.GetPort()
}

// setPortInOptions sets the port in backend-specific options
func (im *instanceManager) setPortInOptions(options *instance.Options, port int) {
	options.BackendOptions.SetPort(port)
}

// EvictLRUInstance finds and stops the least recently used running instance.
func (im *instanceManager) EvictLRUInstance() error {
	return im.lifecycle.EvictLRU()
}
