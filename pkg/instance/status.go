package instance

import (
	"encoding/json"
	"log"
	"sync"
)

// Status is the enum for status values (exported).
type Status int

const (
	Stopped Status = iota
	Running
	Failed
	Restarting
)

var nameToStatus = map[string]Status{
	"stopped":    Stopped,
	"running":    Running,
	"failed":     Failed,
	"restarting": Restarting,
}

var statusToName = map[Status]string{
	Stopped:    "stopped",
	Running:    "running",
	Failed:     "failed",
	Restarting: "restarting",
}

// Status enum JSON marshaling methods
func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := statusToName[s]
	if !ok {
		name = "stopped" // Default to "stopped" for unknown status
	}
	return json.Marshal(name)
}

// UnmarshalJSON implements json.Unmarshaler for Status enum
func (s *Status) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	status, ok := nameToStatus[str]
	if !ok {
		log.Printf("Unknown instance status: %s", str)
		status = Stopped // Default to Stopped on unknown status
	}

	*s = status
	return nil
}

// status represents the instance status with thread-safe access (unexported).
type status struct {
	mu sync.RWMutex
	s  Status

	// Callback for status changes
	onStatusChange func(oldStatus, newStatus Status)
}

// newStatus creates a new status wrapper with the given initial status
func newStatus(initial Status) *status {
	return &status{
		s: initial,
	}
}

// get returns the current status
func (st *status) get() Status {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s
}

// set updates the status and triggers the onStatusChange callback if set
func (st *status) set(newStatus Status) {
	st.mu.Lock()
	oldStatus := st.s
	st.s = newStatus
	callback := st.onStatusChange
	st.mu.Unlock()

	// Call the callback outside the lock to avoid potential deadlocks
	if callback != nil {
		callback(oldStatus, newStatus)
	}
}

// isRunning returns true if the status is Running
func (st *status) isRunning() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s == Running
}

// MarshalJSON implements json.Marshaler for status wrapper
func (st *status) MarshalJSON() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler for status wrapper
func (st *status) UnmarshalJSON(data []byte) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s.UnmarshalJSON(data)
}
