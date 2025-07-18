package llamactl

import (
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type Instance struct {
	ID      uuid.UUID
	Args    []string
	Options *InstanceOptions

	// Status
	Running bool

	// Output channels
	StdOut chan string // Channel for sending output messages
	StdErr chan string // Channel for sending error messages

	// internal
	cmd    *exec.Cmd          // Command to run the instance
	ctx    context.Context    // Context for managing the instance lifecycle
	cancel context.CancelFunc // Function to cancel the context
	stdout io.ReadCloser      // Standard output stream
	stderr io.ReadCloser      // Standard error stream
	mu     sync.Mutex         // Mutex for synchronizing access to the instance
}

func NewInstance(id uuid.UUID, options *InstanceOptions) *Instance {
	return &Instance{
		ID:      id,
		Args:    options.BuildCommandArgs(),
		Options: options,

		Running: false,

		StdOut: make(chan string, 100),
		StdErr: make(chan string, 100),
	}
}

func (i *Instance) Start() *exec.Cmd {
	args := i.Options.BuildCommandArgs()
	cmd := exec.Command("llama-server", args...)

	cmd.Start()
	return cmd
}
