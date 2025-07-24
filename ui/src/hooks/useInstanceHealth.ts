// ui/src/hooks/useInstanceHealth.ts
import { useState, useEffect } from 'react'
import { HealthStatus } from '@/types/instance'
import { healthService } from '@/lib/healthService'

export function useInstanceHealth(instanceName: string, isRunning: boolean): HealthStatus | undefined {
  const [health, setHealth] = useState<HealthStatus | undefined>()

  useEffect(() => {
    if (!isRunning) {
      setHealth(undefined)
      return
    }

    // Subscribe to health updates for this instance
    const unsubscribe = healthService.subscribe(instanceName, (healthStatus) => {
      setHealth(healthStatus)
    })

    // Cleanup subscription on unmount or when running changes
    return unsubscribe
  }, [instanceName, isRunning])

  return health
}