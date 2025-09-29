import { type ReactNode, createContext, useCallback, useContext, useEffect, useState } from 'react'

interface AuthContextState {
  isAuthenticated: boolean
  isLoading: boolean
  apiKey: string | null
  error: string | null
}

interface AuthContextActions {
  login: (apiKey: string) => Promise<void>
  logout: () => void
  clearError: () => void
  validateAuth: () => Promise<boolean>
}

type AuthContextType = AuthContextState & AuthContextActions

const AuthContext = createContext<AuthContextType | undefined>(undefined)

interface AuthProviderProps {
  children: ReactNode
}

const AUTH_STORAGE_KEY = 'llamactl_management_key'

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Load auth state from sessionStorage on mount
  useEffect(() => {
    const loadStoredAuth = async () => {
      try {
        const storedKey = sessionStorage.getItem(AUTH_STORAGE_KEY)
        if (storedKey) {
          setApiKey(storedKey)
          // Validate the stored key
          const isValid = await validateApiKey(storedKey)
          if (isValid) {
            setIsAuthenticated(true)
          } else {
            // Invalid key, remove it
            sessionStorage.removeItem(AUTH_STORAGE_KEY)
            setApiKey(null)
          }
        }
      } catch (err) {
        console.error('Error loading stored auth:', err)
        // Clear potentially corrupted storage
        sessionStorage.removeItem(AUTH_STORAGE_KEY)
      } finally {
        setIsLoading(false)
      }
    }

    void loadStoredAuth()
  }, [])

  // Validate API key by making a test request
  const validateApiKey = async (key: string): Promise<boolean> => {
    try {
      const response = await fetch(document.baseURI + 'api/v1/instances', {
        headers: {
          'Authorization': `Bearer ${key}`,
          'Content-Type': 'application/json'
        }
      })
      
      return response.ok
    } catch (err) {
      console.error('Auth validation error:', err)
      return false
    }
  }

  const login = useCallback(async (key: string) => {
    setIsLoading(true)
    setError(null)

    try {
      // Validate the provided API key
      const isValid = await validateApiKey(key)
      
      if (!isValid) {
        throw new Error('Invalid API key')
      }

      // Store the key and update state
      sessionStorage.setItem(AUTH_STORAGE_KEY, key)
      setApiKey(key)
      setIsAuthenticated(true)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Authentication failed'
      setError(errorMessage)
      throw new Error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }, [])

  const logout = useCallback(() => {
    sessionStorage.removeItem(AUTH_STORAGE_KEY)
    setApiKey(null)
    setIsAuthenticated(false)
    setError(null)
  }, [])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const validateAuth = useCallback(async (): Promise<boolean> => {
    if (!apiKey) return false
    
    const isValid = await validateApiKey(apiKey)
    if (!isValid) {
      logout()
    }
    return isValid
  }, [apiKey, logout])

  const value: AuthContextType = {
    isAuthenticated,
    isLoading,
    apiKey,
    error,
    login,
    logout,
    clearError,
    validateAuth,
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

// Helper hook for getting auth headers
export const useAuthHeaders = (): HeadersInit => {
  const { apiKey, isAuthenticated } = useAuth()
  
  if (!isAuthenticated || !apiKey) {
    return {}
  }
  
  return {
    'Authorization': `Bearer ${apiKey}`
  }
}
