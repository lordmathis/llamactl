package manager

import (
	"fmt"
	"llamactl/pkg/instance"
	"log"
)

func (im *instanceManager) checkAllTimeouts() {
	im.mu.RLock()
	var timeoutInstances []string

	// Identify instances that should timeout
	for _, inst := range im.instances {
		// Skip remote instances - they are managed by their respective nodes
		if inst.IsRemote() {
			continue
		}

		if inst.ShouldTimeout() {
			timeoutInstances = append(timeoutInstances, inst.Name)
		}
	}
	im.mu.RUnlock() // Release read lock before calling StopInstance

	// Stop the timed-out instances
	for _, name := range timeoutInstances {
		log.Printf("Instance %s has timed out, stopping it", name)
		if _, err := im.StopInstance(name); err != nil {
			log.Printf("Error stopping instance %s: %v", name, err)
		} else {
			log.Printf("Instance %s stopped successfully", name)
		}
	}
}

// EvictLRUInstance finds and stops the least recently used running instance.
func (im *instanceManager) EvictLRUInstance() error {
	im.mu.RLock()
	var lruInstance *instance.Process

	for name := range im.runningInstances {
		inst := im.instances[name]
		if inst == nil {
			continue
		}

		// Skip remote instances - they are managed by their respective nodes
		if inst.IsRemote() {
			continue
		}

		if inst.GetOptions() != nil && inst.GetOptions().IdleTimeout != nil && *inst.GetOptions().IdleTimeout <= 0 {
			continue // Skip instances without idle timeout
		}

		if lruInstance == nil {
			lruInstance = inst
		}

		if inst.LastRequestTime() < lruInstance.LastRequestTime() {
			lruInstance = inst
		}
	}
	im.mu.RUnlock()

	if lruInstance == nil {
		return fmt.Errorf("failed to find lru instance")
	}

	// Evict Instance
	_, err := im.StopInstance(lruInstance.Name)
	return err
}
