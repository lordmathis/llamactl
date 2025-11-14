import { type ReactNode, createContext, useCallback, useContext, useEffect, useState } from 'react'
import { serverApi } from '@/lib/api'
import type { AppConfig } from '@/types/config'

interface ConfigContextState {
  config: AppConfig | null
  isLoading: boolean
  error: string | null
}

interface ConfigContextActions {
  refetchConfig: () => Promise<void>
}

type ConfigContextType = ConfigContextState & ConfigContextActions

const ConfigContext = createContext<ConfigContextType | undefined>(undefined)

interface ConfigProviderProps {
  children: ReactNode
}

export const ConfigProvider = ({ children }: ConfigProviderProps) => {
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const data = await serverApi.getConfig()
      setConfig(data)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load configuration'
      setError(errorMessage)
      console.error('Error loading config:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Load config on mount
  useEffect(() => {
    void fetchConfig()
  }, [fetchConfig])

  const refetchConfig = useCallback(async () => {
    await fetchConfig()
  }, [fetchConfig])

  const value: ConfigContextType = {
    config,
    isLoading,
    error,
    refetchConfig,
  }

  return (
    <ConfigContext.Provider value={value}>
      {children}
    </ConfigContext.Provider>
  )
}

export const useConfig = (): ConfigContextType => {
  const context = useContext(ConfigContext)
  if (context === undefined) {
    throw new Error('useConfig must be used within a ConfigProvider')
  }
  return context
}

// Helper hook to get instance default values from config
export const useInstanceDefaults = () => {
  const { config } = useConfig()

  if (!config) {
    return null
  }

  return {
    autoRestart: config.instances.default_auto_restart,
    maxRestarts: config.instances.default_max_restarts,
    restartDelay: config.instances.default_restart_delay,
    onDemandStart: config.instances.default_on_demand_start,
  }
}

// Helper hook to get backend settings from config
export const useBackendConfig = () => {
  const { config } = useConfig()

  if (!config) {
    return null
  }

  return config.backends
}
