import { type ReactNode, createContext, useContext, useEffect, useState, useRef } from 'react'
import { serverApi } from '@/lib/api'
import type { AppConfig } from '@/types/config'
import { useAuth } from './AuthContext'

interface ConfigContextType {
  config: AppConfig | null
  isLoading: boolean
  error: string | null
}

const ConfigContext = createContext<ConfigContextType | undefined>(undefined)

interface ConfigProviderProps {
  children: ReactNode
}

export const ConfigProvider = ({ children }: ConfigProviderProps) => {
  const { isAuthenticated } = useAuth()
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const loadedRef = useRef(false)

  useEffect(() => {
    if (!isAuthenticated || loadedRef.current) {
      setIsLoading(false)
      return
    }

    loadedRef.current = true

    const loadConfig = async () => {
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
    }

    void loadConfig()
  }, [isAuthenticated])

  return (
    <ConfigContext.Provider value={{ config, isLoading, error }}>
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

  if (!config || !config.instances) {
    return null
  }

  return {
    autoRestart: config.instances.default_auto_restart,
    maxRestarts: config.instances.default_max_restarts,
    restartDelay: config.instances.default_restart_delay,
    onDemandStart: config.instances.default_on_demand_start,
  }
}

// Helper hook to get specific backend settings by backend type
export const useBackendSettings = (backendType: string | undefined) => {
  const { config } = useConfig()

  if (!config || !config.backends || !backendType) {
    return null
  }

  // Map backend type to config key
  const backendKey = backendType === 'llama_cpp'
    ? 'llama-cpp'
    : backendType === 'mlx_lm'
    ? 'mlx'
    : backendType === 'vllm'
    ? 'vllm'
    : null

  if (!backendKey) {
    return null
  }

  const backendConfig = config.backends[backendKey as keyof typeof config.backends]

  if (!backendConfig) {
    return null
  }

  return {
    command: backendConfig.command || '',
    dockerEnabled: backendConfig.docker?.enabled ?? false,
    dockerImage: backendConfig.docker?.image || '',
  }
}
