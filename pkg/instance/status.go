package instance

import (
	"encoding/json"
	"log"
)

// Enum for instance status
type Status int

const (
	Stopped Status = iota
	Running
	Failed
)

var nameToStatus = map[string]Status{
	"stopped": Stopped,
	"running": Running,
	"failed":  Failed,
}

var statusToName = map[Status]string{
	Stopped: "stopped",
	Running: "running",
	Failed:  "failed",
}

func (p *Instance) SetStatus(status Status) {
	oldStatus := p.Status
	p.Status = status

	if p.onStatusChange != nil {
		p.onStatusChange(oldStatus, status)
	}
}

func (p *Instance) GetStatus() Status {
	return p.Status
}

// IsRunning returns true if the status is Running
func (p *Instance) IsRunning() bool {
	return p.Status == Running
}

func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := statusToName[s]
	if !ok {
		name = "stopped" // Default to "stopped" for unknown status
	}
	return json.Marshal(name)
}

// UnmarshalJSON implements json.Unmarshaler
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
