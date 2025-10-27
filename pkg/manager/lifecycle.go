package manager

import (
	"fmt"
	"llamactl/pkg/instance"
	"log"
	"sync"
	"time"
)

// lifecycleManager handles background timeout checking and LRU eviction.
// It properly coordinates shutdown to prevent races with the timeout checker.
type lifecycleManager struct {
	registry *instanceRegistry
	manager  InstanceManager // For calling Stop/Evict operations

	ticker        *time.Ticker
	checkInterval time.Duration
	enableLRU     bool

	shutdownChan chan struct{}
	shutdownDone chan struct{}
	shutdownOnce sync.Once
}

// newLifecycleManager creates a new lifecycle manager.
func newLifecycleManager(
	registry *instanceRegistry,
	manager InstanceManager,
	checkInterval time.Duration,
	enableLRU bool,
) *lifecycleManager {
	if checkInterval <= 0 {
		checkInterval = 5 * time.Minute // Default to 5 minutes
	}

	return &lifecycleManager{
		registry:      registry,
		manager:       manager,
		ticker:        time.NewTicker(checkInterval),
		checkInterval: checkInterval,
		enableLRU:     enableLRU,
		shutdownChan:  make(chan struct{}),
		shutdownDone:  make(chan struct{}),
	}
}

// Start begins the timeout checking loop in a goroutine.
func (l *lifecycleManager) start() {
	go l.timeoutCheckLoop()
}

// Stop gracefully stops the lifecycle manager.
// This ensures the timeout checker completes before instance cleanup begins.
func (l *lifecycleManager) stop() {
	l.shutdownOnce.Do(func() {
		close(l.shutdownChan)
		<-l.shutdownDone // Wait for checker to finish (prevents shutdown race)
		l.ticker.Stop()
	})
}

// timeoutCheckLoop is the main loop that periodically checks for timeouts.
func (l *lifecycleManager) timeoutCheckLoop() {
	defer close(l.shutdownDone) // Signal completion

	for {
		select {
		case <-l.ticker.C:
			l.checkTimeouts()
		case <-l.shutdownChan:
			return // Exit goroutine on shutdown
		}
	}
}

// checkTimeouts checks all instances for timeout and stops those that have timed out.
func (l *lifecycleManager) checkTimeouts() {
	// Get all instances from registry
	instances := l.registry.list()

	var timeoutInstances []string

	// Identify instances that should timeout
	for _, inst := range instances {
		// Skip remote instances - they are managed by their respective nodes
		if inst.IsRemote() {
			continue
		}

		// Only check running instances
		if !l.registry.isRunning(inst.Name) {
			continue
		}

		if inst.ShouldTimeout() {
			timeoutInstances = append(timeoutInstances, inst.Name)
		}
	}

	// Stop the timed-out instances
	for _, name := range timeoutInstances {
		log.Printf("Instance %s has timed out, stopping it", name)
		if _, err := l.manager.StopInstance(name); err != nil {
			log.Printf("Error stopping instance %s: %w", name, err)
		} else {
			log.Printf("Instance %s stopped successfully", name)
		}
	}
}

// EvictLRU finds and stops the least recently used running instance.
// This is called when max running instances limit is reached.
func (l *lifecycleManager) evictLRU() error {
	if !l.enableLRU {
		return fmt.Errorf("LRU eviction is not enabled")
	}

	// Get all running instances
	runningInstances := l.registry.listRunning()

	var lruInstance *instance.Instance

	for _, inst := range runningInstances {
		// Skip remote instances - they are managed by their respective nodes
		if inst.IsRemote() {
			continue
		}

		// Skip instances without idle timeout
		if inst.GetOptions() != nil && inst.GetOptions().IdleTimeout != nil && *inst.GetOptions().IdleTimeout <= 0 {
			continue
		}

		if lruInstance == nil {
			lruInstance = inst
		}

		if inst.LastRequestTime() < lruInstance.LastRequestTime() {
			lruInstance = inst
		}
	}

	if lruInstance == nil {
		return fmt.Errorf("failed to find lru instance")
	}

	// Evict the LRU instance
	log.Printf("Evicting LRU instance %s", lruInstance.Name)
	_, err := l.manager.StopInstance(lruInstance.Name)
	return err
}
