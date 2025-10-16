package instance

// UpdateLastRequestTime updates the last request access time for the instance via proxy
// Delegates to the Proxy component
func (i *Instance) UpdateLastRequestTime() {
	if i.proxy != nil {
		i.proxy.UpdateLastRequestTime()
	}
}

// ShouldTimeout checks if the instance should timeout based on idle time
// Delegates to the Proxy component
func (i *Instance) ShouldTimeout() bool {
	if i.proxy == nil {
		return false
	}
	return i.proxy.ShouldTimeout()
}
