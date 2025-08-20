package instance

// UpdateLastRequestTime updates the last request access time for the instance via proxy
func (i *Process) UpdateLastRequestTime() {
	i.mu.Lock()
	defer i.mu.Unlock()

	lastRequestTime := i.timeProvider.Now().Unix()
	i.lastRequestTime.Store(lastRequestTime)
}

func (i *Process) ShouldTimeout() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.Running || i.options.IdleTimeout == nil || *i.options.IdleTimeout <= 0 {
		return false
	}

	// Check if the last request time exceeds the idle timeout
	lastRequest := i.lastRequestTime.Load()
	idleTimeoutMinutes := *i.options.IdleTimeout

	// Convert timeout from minutes to seconds for comparison
	idleTimeoutSeconds := int64(idleTimeoutMinutes * 60)

	return (i.timeProvider.Now().Unix() - lastRequest) > idleTimeoutSeconds
}
