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
