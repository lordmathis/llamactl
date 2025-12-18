package manager

import (
	"fmt"
	"llamactl/pkg/instance"
	"sync"
)

// modelRegistry maintains a global mapping of model names to instance names
// for llama.cpp instances. Model names must be globally unique across all instances.
type modelRegistry struct {
	mu              sync.RWMutex
	modelToInstance map[string]string   // model name → instance name
	instanceModels  map[string][]string // instance name → model names
}

// newModelRegistry creates a new model registry
func newModelRegistry() *modelRegistry {
	return &modelRegistry{
		modelToInstance: make(map[string]string),
		instanceModels:  make(map[string][]string),
	}
}

// registerModels registers models from an instance to the registry.
// Skips models that conflict with other instances and returns a list of conflicts.
func (mr *modelRegistry) registerModels(instanceName string, models []instance.Model) []string {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Unregister any existing models for this instance first
	mr.removeModels(instanceName)

	// Register models, skipping conflicts
	var modelNames []string
	var conflicts []string

	for _, model := range models {
		// Check if this model conflicts with another instance
		if existingInstance, exists := mr.modelToInstance[model.ID]; exists && existingInstance != instanceName {
			conflicts = append(conflicts, fmt.Sprintf("%s (already in %s)", model.ID, existingInstance))
			continue // Skip this model
		}

		// Register the model
		mr.modelToInstance[model.ID] = instanceName
		modelNames = append(modelNames, model.ID)
	}

	mr.instanceModels[instanceName] = modelNames

	return conflicts
}

// unregisterModels removes all models for an instance
func (mr *modelRegistry) unregisterModels(instanceName string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.removeModels(instanceName)
}

// removeModels removes all models for an instance (caller must hold lock)
func (mr *modelRegistry) removeModels(instanceName string) {
	if models, exists := mr.instanceModels[instanceName]; exists {
		for _, modelName := range models {
			delete(mr.modelToInstance, modelName)
		}
		delete(mr.instanceModels, instanceName)
	}
}

// getModelInstance returns the instance name that hosts the given model
func (mr *modelRegistry) getModelInstance(modelName string) (string, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	instanceName, exists := mr.modelToInstance[modelName]
	return instanceName, exists
}
