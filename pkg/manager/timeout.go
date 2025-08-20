package manager

import "log"

func (im *instanceManager) checkAllTimeouts() {
	im.mu.RLock()
	var timeoutInstances []string

	// Identify instances that should timeout
	for _, inst := range im.instances {
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
