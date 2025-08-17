package instance

import "time"

// UpdateLastRequestTime updates the last request access time for the instance via proxy
func (i *Process) UpdateLastRequestTime() {
	i.mu.Lock()
	defer i.mu.Unlock()

	lastRequestTime := time.Now().Unix()
	i.lastRequestTime.Store(lastRequestTime)
}

func (i *Process) ShouldTimeout() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// If idle timeout is not set, no timeout
	if i.options.IdleTimeout == nil || *i.options.IdleTimeout <= 0 {
		return false
	}

	// Check if the last request time exceeds the idle timeout
	lastRequest := i.lastRequestTime.Load()
	idleTimeout := *i.options.IdleTimeout

	return (time.Now().Unix() - lastRequest) > int64(idleTimeout)
}
