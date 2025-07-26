import type { ReactNode } from 'react';
import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { CreateInstanceOptions, Instance } from '@/types/instance'
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
  const [instancesMap, setInstancesMap] = useState<Map<string, Instance>>(new Map())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Convert map to array for consumers
  const instances = Array.from(instancesMap.values())

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const fetchInstances = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await instancesApi.list()
      
      // Convert array to map
      const newMap = new Map<string, Instance>()
      data.forEach(instance => {
        newMap.set(instance.name, instance)
      })
      setInstancesMap(newMap)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch instances')
    } finally {
      setLoading(false)
    }
  }, [])

  const updateInstanceInMap = useCallback((name: string, updates: Partial<Instance>) => {
    setInstancesMap(prev => {
      const newMap = new Map(prev)
      const existing = newMap.get(name)
      if (existing) {
        newMap.set(name, { ...existing, ...updates })
      }
      return newMap
    })
  }, [])

  const createInstance = useCallback(async (name: string, options: CreateInstanceOptions) => {
    try {
      setError(null)
      const newInstance = await instancesApi.create(name, options)
      
      // Add to map directly
      setInstancesMap(prev => {
        const newMap = new Map(prev)
        newMap.set(name, newInstance)
        return newMap
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create instance')
    }
  }, [])

  const updateInstance = useCallback(async (name: string, options: CreateInstanceOptions) => {
    try {
      setError(null)
      const updatedInstance = await instancesApi.update(name, options)
      
      // Update in map directly
      setInstancesMap(prev => {
        const newMap = new Map(prev)
        newMap.set(name, updatedInstance)
        return newMap
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update instance')
    }
  }, [])

  const startInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.start(name)
      
      // Update only this instance's running status
      updateInstanceInMap(name, { running: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start instance')
    }
  }, [updateInstanceInMap])

  const stopInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.stop(name)
      
      // Update only this instance's running status
      updateInstanceInMap(name, { running: false })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop instance')
    }
  }, [updateInstanceInMap])

  const restartInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.restart(name)
      
      // Update only this instance's running status
      updateInstanceInMap(name, { running: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to restart instance')
    }
  }, [updateInstanceInMap])

  const deleteInstance = useCallback(async (name: string) => {
    try {
      setError(null)
      await instancesApi.delete(name)
      
      // Remove from map directly
      setInstancesMap(prev => {
        const newMap = new Map(prev)
        newMap.delete(name)
        return newMap
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete instance')
    }
  }, [])

  useEffect(() => {
    fetchInstances()
  }, [fetchInstances])

  const value: InstancesContextType = {
    instances,
    loading,
    error,
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