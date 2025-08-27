// ui/src/hooks/useInstanceHealth.ts
import { useState, useEffect } from 'react'
import type { HealthStatus, InstanceStatus } from '@/types/instance'
import { healthService } from '@/lib/healthService'

export function useInstanceHealth(instanceName: string, instanceStatus: InstanceStatus): HealthStatus | undefined {
  const [health, setHealth] = useState<HealthStatus | undefined>()

  useEffect(() => {
    if (instanceStatus == "stopped") {
      setHealth({ status: "unknown", lastChecked: new Date() })
      return
    }
      
    if  (instanceStatus == "failed") {
      setHealth({ status: instanceStatus, lastChecked: new Date() })
      return
    }

    // Subscribe to health updates for this instance
    const unsubscribe = healthService.subscribe(instanceName, (healthStatus) => {
      setHealth(healthStatus)
    })

    // Cleanup subscription on unmount or when instanceStatus changes
    return unsubscribe
  }, [instanceName, instanceStatus])

  return health
}