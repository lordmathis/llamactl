package llamactl

type Instance struct {
	Status  string
	Options *LlamaServerOptions
}

type InstanceManager interface {
}

type instanceManager struct {
	instances map[string]*Instance
}

func NewInstanceManager() InstanceManager {
	return &instanceManager{
		instances: make(map[string]*Instance),
	}
}
