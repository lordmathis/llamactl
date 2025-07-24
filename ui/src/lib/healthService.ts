// ui/src/lib/healthService.ts
import { HealthStatus } from '@/types/instance'

class HealthService {
  private intervals: Map<string, NodeJS.Timeout> = new Map()
  private startupTimeouts: Map<string, NodeJS.Timeout> = new Map()

  async checkHealth(instanceName: string): Promise<HealthStatus> {
    try {
      const response = await fetch(`/api/v1/instances/${instanceName}/proxy/health`)
      
      if (response.status === 200) {
        return {
          status: 'ok',
          lastChecked: new Date()
        }
      } else if (response.status === 503) {
        const data = await response.json()
        return {
          status: 'loading',
          message: data.error.message,
          lastChecked: new Date()
        }
      } else {
        return {
          status: 'error',
          message: `HTTP ${response.status}`,
          lastChecked: new Date()
        }
      }
    } catch (error) {
      return {
        status: 'error',
        message: 'Network error',
        lastChecked: new Date()
      }
    }
  }

  startHealthCheck(instanceName: string, onUpdate: (health: HealthStatus) => void): void {
    // Don't start if already checking
    if (this.isChecking(instanceName)) {
      return
    }
    
    const startupTimeout = setTimeout(() => {
      this.startupTimeouts.delete(instanceName)
      
      const check = async () => {
        const health = await this.checkHealth(instanceName)
        onUpdate(health)
      }
      
      check()
      const interval = setInterval(check, 60000)
      this.intervals.set(instanceName, interval)
    }, 2000)
    
    this.startupTimeouts.set(instanceName, startupTimeout)
  }

  stopHealthCheck(instanceName: string): void {
    // Clear startup timeout if exists
    const startupTimeout = this.startupTimeouts.get(instanceName)
    if (startupTimeout) {
      clearTimeout(startupTimeout)
      this.startupTimeouts.delete(instanceName)
    }
    
    // Clear interval if exists
    const interval = this.intervals.get(instanceName)
    if (interval) {
      clearInterval(interval)
      this.intervals.delete(instanceName)
    }
  }

  stopAll(): void {
    this.startupTimeouts.forEach(timeout => clearTimeout(timeout))
    this.startupTimeouts.clear()
    this.intervals.forEach(interval => clearInterval(interval))
    this.intervals.clear()
  }

  isChecking(instanceName: string): boolean {
    return this.intervals.has(instanceName) || this.startupTimeouts.has(instanceName)
  }
}

export const healthService = new HealthService()