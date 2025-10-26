// ui/src/hooks/useInstanceHealth.ts
import { useState, useEffect } from 'react'
import type { HealthStatus, InstanceStatus } from '@/types/instance'
import { healthService } from '@/lib/healthService'

export function useInstanceHealth(instanceName: string, instanceStatus: InstanceStatus): HealthStatus | undefined {
  const [health, setHealth] = useState<HealthStatus | undefined>()

  useEffect(() => {
    // Subscribe to health updates for this instance
    const unsubscribe = healthService.subscribe(instanceName, (healthStatus) => {
      setHealth(healthStatus)
    })

    // Cleanup subscription on unmount or when instance changes
    return unsubscribe
  }, [instanceName])

  // Trigger health check when instance status changes to active states
  useEffect(() => {
    if (instanceStatus === 'running' || instanceStatus === 'restarting') {
      healthService.refreshHealth(instanceName).catch(error => {
        console.error(`Failed to refresh health for ${instanceName}:`, error)
      })
    }
  }, [instanceName, instanceStatus])

  return health
}
