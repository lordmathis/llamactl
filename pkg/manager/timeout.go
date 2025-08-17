package manager

import "log"

func (im *instanceManager) checkAllTimeouts() {
	im.mu.RLock()
	defer im.mu.RUnlock()

	for _, inst := range im.instances {
		if inst.ShouldTimeout() {
			log.Printf("Instance %s has timed out, stopping it", inst.Name)
			if proc, err := im.StopInstance(inst.Name); err != nil {
				log.Printf("Error stopping instance %s: %v", inst.Name, err)
			} else {
				log.Printf("Instance %s stopped successfully, process: %v", inst.Name, proc)
			}
		}
	}
}
