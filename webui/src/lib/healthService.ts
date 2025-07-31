import { type HealthStatus } from '@/types/instance'
import { instancesApi } from '@/lib/api'

type HealthCallback = (health: HealthStatus) => void

class HealthService {
  private intervals: Map<string, NodeJS.Timeout> = new Map()
  private callbacks: Map<string, Set<HealthCallback>> = new Map()

  async checkHealth(instanceName: string): Promise<HealthStatus> {
    try {
      await instancesApi.getHealth(instanceName)
      
      return {
        status: 'ok',
        lastChecked: new Date()
      }
    } catch (error) {
      if (error instanceof Error) {
        // Check if it's a 503 (service unavailable - loading)
        if (error.message.includes('503')) {
          return {
            status: 'loading',
            message: 'Instance is starting up',
            lastChecked: new Date()
          }
        }
        
        return {
          status: 'error',
          message: error.message,
          lastChecked: new Date()
        }
      }
      
      return {
        status: 'error',
        message: 'Unknown error',
        lastChecked: new Date()
      }
    }
  }

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
        }
      }
    }
  }

  private startHealthCheck(instanceName: string): void {
    if (this.intervals.has(instanceName)) {
      return // Already checking
    }

    // Initial check with delay
    setTimeout(async () => {
      const health = await this.checkHealth(instanceName)
      this.notifyCallbacks(instanceName, health)
      
      // Start periodic checks
      const interval = setInterval(async () => {
        const health = await this.checkHealth(instanceName)
        this.notifyCallbacks(instanceName, health)
      }, 60000)
      
      this.intervals.set(instanceName, interval)
    }, 5000)
  }

  private stopHealthCheck(instanceName: string): void {
    const interval = this.intervals.get(instanceName)
    if (interval) {
      clearInterval(interval)
      this.intervals.delete(instanceName)
    }
  }

  private notifyCallbacks(instanceName: string, health: HealthStatus): void {
    const callbacks = this.callbacks.get(instanceName)
    if (callbacks) {
      callbacks.forEach(callback => callback(health))
    }
  }

  stopAll(): void {
    this.intervals.forEach(interval => clearInterval(interval))
    this.intervals.clear()
    this.callbacks.clear()
  }
}

export const healthService = new HealthService()

// Export the individual checkHealth function as well
export async function checkHealth(instanceName: string): Promise<HealthStatus> {
  return healthService.checkHealth(instanceName)
}