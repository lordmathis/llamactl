import { useState, useEffect, useCallback } from 'react'
import { Instance } from '@/types/instance'
import { instancesApi } from '@/lib/api'

interface UseInstancesState {
  instances: Instance[]
  loading: boolean
  error: string | null
}

interface UseInstancesActions {
  fetchInstances: () => Promise<void>
  startInstance: (name: string) => Promise<void>
  stopInstance: (name: string) => Promise<void>
  restartInstance: (name: string) => Promise<void>
  deleteInstance: (name: string) => Promise<void>
  clearError: () => void
}

export const useInstances = (): UseInstancesState & UseInstancesActions => {
  const [instances, setInstances] = useState<Instance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const fetchInstances = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await instancesApi.list()
      setInstances(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch instances')
    } finally {
      setLoading(false)
    }
  }, [])

  const startInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.start(name)
      // Refresh the list to get updated status
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start instance')
    }
  }, [fetchInstances])

  const stopInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.stop(name)
      // Refresh the list to get updated status
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop instance')
    }
  }, [fetchInstances])

  const restartInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.restart(name)
      // Refresh the list to get updated status
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to restart instance')
    }
  }, [fetchInstances])

  const deleteInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.delete(name)
      // Refresh the list to get updated status
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete instance')
    }
  }, [fetchInstances])

  // Fetch instances on mount
  useEffect(() => {
    fetchInstances()
  }, [fetchInstances])

  return {
    // State
    instances,
    loading,
    error,
    // Actions
    fetchInstances,
    startInstance,
    stopInstance,
    restartInstance,
    deleteInstance,
    clearError,
  }
}