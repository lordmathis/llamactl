import { type ReactNode, createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { CachedModel, DownloadJob } from '@/types/model'
import { llamaCppModelsApi } from '@/lib/api'
import { useAuth } from '@/contexts/AuthContext'
import { modelsPollingService } from '@/lib/modelsPollingService'

interface ModelsContextType {
  // State
  models: CachedModel[]
  activeJobs: DownloadJob[]
  loading: boolean
  error: string | null

  // Actions
  fetchModels: () => Promise<void>
  startDownload: (repo: string, tag?: string) => Promise<void>
  cancelDownload: (jobId: string) => Promise<void>
  deleteModel: (repo: string, tag?: string) => Promise<void>
  clearError: () => void
}

const ModelsContext = createContext<ModelsContextType | undefined>(undefined)

interface ModelsProviderProps {
  children: ReactNode
}

export const ModelsProvider = ({ children }: ModelsProviderProps) => {
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const [models, setModels] = useState<CachedModel[]>([])
  const [activeJobs, setActiveJobs] = useState<DownloadJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const fetchModels = useCallback(async () => {
    if (!isAuthenticated) {
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)
      const modelsData = await llamaCppModelsApi.listModels()
      setModels(modelsData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch models')
    } finally {
      setLoading(false)
    }
  }, [isAuthenticated])

  const startDownload = useCallback(async (repo: string, tag?: string) => {
    setError(null)
    try {
      await llamaCppModelsApi.startDownload(repo, tag)
      // Notify polling service that we're expecting a new job
      modelsPollingService.downloadStarted()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to start download'
      setError(message)
      throw err
    }
  }, [])

  const cancelDownload = useCallback(async (jobId: string) => {
    setError(null)
    try {
      await llamaCppModelsApi.cancelJob(jobId)
      // Polling will automatically pick up the status change
      // Refresh models list (in case partial files were deleted)
      await fetchModels()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to cancel download'
      setError(message)
      throw err
    }
  }, [fetchModels])

  const deleteModel = useCallback(async (repo: string, tag?: string) => {
    setError(null)
    try {
      await llamaCppModelsApi.deleteModel(repo, tag)
      // Refresh models list after deletion
      await fetchModels()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete model'
      setError(message)
      throw err
    }
  }, [fetchModels])

  // Subscribe to polling service updates
  useEffect(() => {
    if (!isAuthenticated) {
      return
    }

    let previousActiveCount = 0

    const unsubscribe = modelsPollingService.subscribe((jobs) => {
      setActiveJobs(jobs)

      // If we just completed downloads (went from active to no active jobs)
      // refresh the models list to show newly downloaded models
      if (previousActiveCount > 0 && jobs.length === 0) {
        void fetchModels()
      }

      previousActiveCount = jobs.length
    })

    return () => {
      unsubscribe()
    }
  }, [isAuthenticated, fetchModels])

  // Fetch models when auth is ready and user is authenticated
  useEffect(() => {
    if (!authLoading) {
      if (isAuthenticated) {
        void fetchModels()
      } else {
        // Clear state when not authenticated
        setModels([])
        setActiveJobs([])
        setLoading(false)
        setError(null)
      }
    }
  }, [authLoading, isAuthenticated, fetchModels])

  const value: ModelsContextType = {
    models,
    activeJobs,
    loading,
    error,
    fetchModels,
    startDownload,
    cancelDownload,
    deleteModel,
    clearError,
  }

  return (
    <ModelsContext.Provider value={value}>
      {children}
    </ModelsContext.Provider>
  )
}

export const useModels = (): ModelsContextType => {
  const context = useContext(ModelsContext)
  if (context === undefined) {
    throw new Error('useModels must be used within a ModelsProvider')
  }
  return context
}
