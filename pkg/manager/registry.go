package manager

import (
	"fmt"
	"llamactl/pkg/instance"
	"sync"
)

// instanceRegistry provides thread-safe storage and lookup of instances
// with running state tracking using lock-free sync.Map for status checks.
type instanceRegistry struct {
	mu        sync.RWMutex
	instances map[string]*instance.Instance
	running   sync.Map // map[string]struct{} - lock-free for status checks
}

// NewInstanceRegistry creates a new instance registry.
func NewInstanceRegistry() *instanceRegistry {
	return &instanceRegistry{
		instances: make(map[string]*instance.Instance),
	}
}

// Get retrieves an instance by name.
// Returns the instance and true if found, nil and false otherwise.
func (r *instanceRegistry) Get(name string) (*instance.Instance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, exists := r.instances[name]
	return inst, exists
}

// List returns a snapshot copy of all instances to prevent external mutation.
func (r *instanceRegistry) List() []*instance.Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*instance.Instance, 0, len(r.instances))
	for _, inst := range r.instances {
		result = append(result, inst)
	}
	return result
}

// ListRunning returns a snapshot of all currently running instances.
func (r *instanceRegistry) ListRunning() []*instance.Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*instance.Instance, 0)
	for name, inst := range r.instances {
		if _, isRunning := r.running.Load(name); isRunning {
			result = append(result, inst)
		}
	}
	return result
}

// Add adds a new instance to the registry.
// Returns an error if an instance with the same name already exists.
func (r *instanceRegistry) Add(inst *instance.Instance) error {
	if inst == nil {
		return fmt.Errorf("cannot add nil instance")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[inst.Name]; exists {
		return fmt.Errorf("instance %s already exists", inst.Name)
	}

	r.instances[inst.Name] = inst

	// Initialize running state if the instance is running
	if inst.IsRunning() {
		r.running.Store(inst.Name, struct{}{})
	}

	return nil
}

// Remove removes an instance from the registry.
// Returns an error if the instance doesn't exist.
func (r *instanceRegistry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[name]; !exists {
		return fmt.Errorf("instance %s not found", name)
	}

	delete(r.instances, name)
	r.running.Delete(name)

	return nil
}

// MarkRunning marks an instance as running using lock-free sync.Map.
func (r *instanceRegistry) MarkRunning(name string) {
	r.running.Store(name, struct{}{})
}

// MarkStopped marks an instance as stopped using lock-free sync.Map.
func (r *instanceRegistry) MarkStopped(name string) {
	r.running.Delete(name)
}

// IsRunning checks if an instance is running using lock-free sync.Map.
func (r *instanceRegistry) IsRunning(name string) bool {
	_, isRunning := r.running.Load(name)
	return isRunning
}

// Count returns the total number of instances in the registry.
func (r *instanceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.instances)
}

// CountRunning returns the number of currently running instances.
func (r *instanceRegistry) CountRunning() int {
	count := 0
	r.running.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
