import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react'
import { CreateInstanceOptions, Instance } from '@/types/instance'
import { instancesApi } from '@/lib/api'

interface InstancesContextState {
  instances: Instance[]
  loading: boolean
  error: string | null
}

interface InstancesContextActions {
  fetchInstances: () => Promise<void>
  createInstance: (name: string, options: CreateInstanceOptions) => Promise<void>
  updateInstance: (name: string, options: CreateInstanceOptions) => Promise<void>
  startInstance: (name: string) => Promise<void>
  stopInstance: (name: string) => Promise<void>
  restartInstance: (name: string) => Promise<void>
  deleteInstance: (name: string) => Promise<void>
  clearError: () => void
}

type InstancesContextType = InstancesContextState & InstancesContextActions

const InstancesContext = createContext<InstancesContextType | undefined>(undefined)

interface InstancesProviderProps {
  children: ReactNode
}

export const InstancesProvider = ({ children }: InstancesProviderProps) => {
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

  const createInstance = useCallback(async (name: string, options: CreateInstanceOptions) => {
    try {
      setError(null)
      await instancesApi.create(name, options)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create instance')
    }
  }, [fetchInstances])

  const updateInstance = useCallback(async (name: string, options: CreateInstanceOptions) => {
    try {
      setError(null)
      await instancesApi.update(name, options)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update instance')
    }
  }, [fetchInstances])

  const startInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.start(name)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start instance')
    }
  }, [fetchInstances])

  const stopInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.stop(name)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop instance')
    }
  }, [fetchInstances])

  const restartInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.restart(name)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to restart instance')
    }
  }, [fetchInstances])

  const deleteInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.delete(name)
      await fetchInstances()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete instance')
    }
  }, [fetchInstances])

  // Fetch instances on mount
  useEffect(() => {
    fetchInstances()
  }, [fetchInstances])

  const value: InstancesContextType = {
    // State
    instances,
    loading,
    error,
    // Actions
    fetchInstances,
    createInstance,
    updateInstance,
    startInstance,
    stopInstance,
    restartInstance,
    deleteInstance,
    clearError,
  }

  return (
    <InstancesContext.Provider value={value}>
      {children}
    </InstancesContext.Provider>
  )
}

export const useInstances = (): InstancesContextType => {
  const context = useContext(InstancesContext)
  if (context === undefined) {
    throw new Error('useInstances must be used within an InstancesProvider')
  }
  return context
}
