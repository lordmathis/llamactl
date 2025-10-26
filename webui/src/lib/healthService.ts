import { type HealthStatus, type InstanceStatus, type HealthState } from '@/types/instance'
import { instancesApi } from '@/lib/api'

type HealthCallback = (health: HealthStatus) => void

// Polling intervals based on health state (in milliseconds)
const POLLING_INTERVALS: Record<HealthState, number> = {
  'starting': 5000,    // 5 seconds - frequent during startup
  'restarting': 5000,  // 5 seconds - restart in progress
  'ready': 60000,      // 60 seconds - stable state
  'stopped': 0,        // No polling
  'failed': 0,         // No polling
}

class HealthService {
  private intervals: Map<string, NodeJS.Timeout> = new Map()
  private callbacks: Map<string, Set<HealthCallback>> = new Map()
  private lastHealthState: Map<string, HealthState> = new Map()
  private healthCache: Map<string, { health: HealthStatus; timestamp: number }> = new Map()
  private readonly CACHE_TTL = 2000 // 2 seconds cache

  /**
   * Performs a two-tier health check:
   * 1. Get instance status from backend (authoritative)
   * 2. If running, perform HTTP health check
   */
  async performHealthCheck(instanceName: string): Promise<HealthStatus> {
    // Check cache first
    const cached = this.healthCache.get(instanceName)
    if (cached && Date.now() - cached.timestamp < this.CACHE_TTL) {
      return cached.health
    }

    try {
      // Step 1: Get instance details (includes status)
      const instance = await instancesApi.get(instanceName)

      // Step 2: If running, attempt HTTP health check
      if (instance.status === 'running') {
        try {
          await instancesApi.getHealth(instanceName)

          // HTTP health check succeeded - instance is ready
          const health: HealthStatus = {
            state: 'ready',
            instanceStatus: 'running',
            lastChecked: new Date(),
            source: 'http'
          }

          this.updateCache(instanceName, health)
          return health

        } catch (httpError) {
          // HTTP health check failed - instance is still starting
          // Any error (503, connection refused, timeout, etc.) means "starting"
          const health: HealthStatus = {
            state: 'starting',
            instanceStatus: 'running',
            lastChecked: new Date(),
            error: httpError instanceof Error ? httpError.message : 'Health check failed',
            source: 'http'
          }

          this.updateCache(instanceName, health)
          return health
        }
      } else {
        // Instance not running - map backend status directly
        const health: HealthStatus = {
          state: this.mapStatusToHealthState(instance.status),
          instanceStatus: instance.status,
          lastChecked: new Date(),
          source: 'backend'
        }

        this.updateCache(instanceName, health)
        return health
      }

    } catch (error) {
      // Failed to get instance status from backend
      // This is a backend communication error, not an instance health error
      // Let the error propagate so polling can retry
      console.error(`Failed to get instance status for ${instanceName}:`, error)
      throw error
    }
  }

  /**
   * Maps backend instance status to health state
   */
  private mapStatusToHealthState(status: InstanceStatus): HealthState {
    switch (status) {
      case 'stopped': return 'stopped'
      case 'running': return 'starting' // Should not happen as we check HTTP for running
      case 'failed': return 'failed'
      case 'restarting': return 'restarting'
    }
  }

  /**
   * Updates health cache
   */
  private updateCache(instanceName: string, health: HealthStatus): void {
    this.healthCache.set(instanceName, {
      health,
      timestamp: Date.now()
    })
  }

  /**
   * Manually refresh health for an instance
   */
  async refreshHealth(instanceName: string): Promise<void> {
    // Invalidate cache
    this.healthCache.delete(instanceName)

    try {
      const health = await this.performHealthCheck(instanceName)
      this.notifyCallbacks(instanceName, health)

      // Update last state and adjust polling interval if needed
      const previousState = this.lastHealthState.get(instanceName)
      this.lastHealthState.set(instanceName, health.state)

      if (previousState !== health.state) {
        this.adjustPollingInterval(instanceName, health.state)
      }
    } catch (error) {
      // Error getting health - keep polling if active
      console.error(`Failed to refresh health for ${instanceName}:`, error)
    }
  }

  /**
   * Trigger health check after instance operation
   */
  checkHealthAfterOperation(instanceName: string, operation: 'start' | 'stop' | 'restart'): void {
    // Invalidate cache immediately
    this.healthCache.delete(instanceName)

    // Perform immediate health check
    this.refreshHealth(instanceName).catch(error => {
      console.error(`Failed to check health after ${operation}:`, error)
    })
  }

  /**
   * Subscribe to health updates for an instance
   */
  subscribe(instanceName: string, callback: HealthCallback): () => void {
    if (!this.callbacks.has(instanceName)) {
      this.callbacks.set(instanceName, new Set())
    }

    this.callbacks.get(instanceName)!.add(callback)

    // Start health checking if this is the first subscriber
    if (this.callbacks.get(instanceName)!.size === 1) {
      this.startHealthCheck(instanceName)
    }

    // Return unsubscribe function
    return () => {
      const callbacks = this.callbacks.get(instanceName)
      if (callbacks) {
        callbacks.delete(callback)

        // Stop health checking if no more subscribers
        if (callbacks.size === 0) {
          this.stopHealthCheck(instanceName)
          this.callbacks.delete(instanceName)
          this.lastHealthState.delete(instanceName)
          this.healthCache.delete(instanceName)
        }
      }
    }
  }

  /**
   * Start health checking for an instance
   */
  private startHealthCheck(instanceName: string): void {
    if (this.intervals.has(instanceName)) {
      return // Already checking
    }

    // Initial check immediately
    this.refreshHealth(instanceName).then(() => {
      const currentState = this.lastHealthState.get(instanceName)
      if (currentState) {
        this.adjustPollingInterval(instanceName, currentState)
      }
    }).catch(error => {
      console.error(`Failed to start health check for ${instanceName}:`, error)
    })
  }

  /**
   * Adjust polling interval based on current health state
   */
  private adjustPollingInterval(instanceName: string, state: HealthState): void {
    // Clear existing interval
    this.stopHealthCheck(instanceName)

    const pollInterval = POLLING_INTERVALS[state]

    // Don't poll for stable states (stopped, failed)
    if (pollInterval === 0) {
      return
    }

    // Start new interval with appropriate timing
    const interval = setInterval(async () => {
      try {
        const health = await this.performHealthCheck(instanceName)
        this.notifyCallbacks(instanceName, health)

        // Check if state changed and adjust interval
        const previousState = this.lastHealthState.get(instanceName)
        this.lastHealthState.set(instanceName, health.state)

        if (previousState !== health.state) {
          this.adjustPollingInterval(instanceName, health.state)
        }
      } catch (error) {
        console.error(`Health check failed for ${instanceName}:`, error)
        // Continue polling even on error
      }
    }, pollInterval)

    this.intervals.set(instanceName, interval)
  }

  /**
   * Stop health checking for an instance
   */
  private stopHealthCheck(instanceName: string): void {
    const interval = this.intervals.get(instanceName)
    if (interval) {
      clearInterval(interval)
      this.intervals.delete(instanceName)
    }
  }

  /**
   * Notify all callbacks with health update
   */
  private notifyCallbacks(instanceName: string, health: HealthStatus): void {
    const callbacks = this.callbacks.get(instanceName)
    if (callbacks) {
      callbacks.forEach(callback => callback(health))
    }
  }

  /**
   * Stop all health checking and cleanup
   */
  destroy(): void {
    this.intervals.forEach(interval => clearInterval(interval))
    this.intervals.clear()
    this.callbacks.clear()
    this.lastHealthState.clear()
    this.healthCache.clear()
  }
}

export const healthService = new HealthService()

// Export the individual performHealthCheck function as well
export async function checkHealth(instanceName: string): Promise<HealthStatus> {
  return healthService.performHealthCheck(instanceName)
}
