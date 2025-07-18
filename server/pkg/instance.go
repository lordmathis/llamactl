package llamactl

import (
	"github.com/google/uuid"
)

type Instance struct {
	ID      uuid.UUID
	Port    int
	Status  string
	Options *InstanceOptions
}
