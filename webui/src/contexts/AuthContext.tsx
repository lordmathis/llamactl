import { type ReactNode, createContext, useCallback, useContext, useEffect, useState } from 'react'
import { authApi, type WhoamiResponse, type WhoamiUser } from '@/lib/api'

type AuthMethod = 'key' | 'session' | null

interface AuthContextState {
  isAuthenticated: boolean
  isLoading: boolean
  apiKey: string | null
  user: WhoamiUser | null
  authMethod: AuthMethod
  oidcEnabled: boolean
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
  const [user, setUser] = useState<WhoamiUser | null>(null)
  const [authMethod, setAuthMethod] = useState<AuthMethod>(null)
  const [oidcEnabled, setOidcEnabled] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Validate API key by making a test request
  const validateApiKey = useCallback(async (key: string): Promise<boolean> => {
    try {
      const response = await fetch(`${document.baseURI}api/v1/instances`, {
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
  }, [])

  // Load auth state on mount: stored API key first, then an existing OIDC
  // session cookie
  useEffect(() => {
    const loadStoredAuth = async () => {
      let whoami: WhoamiResponse | null = null
      try {
        whoami = await authApi.whoami()
        setOidcEnabled(whoami.oidc_enabled)
      } catch (err) {
        console.error('Error fetching auth state:', err)
      }

      try {
        const storedKey = sessionStorage.getItem(AUTH_STORAGE_KEY)
        if (storedKey) {
          const isValid = await validateApiKey(storedKey)
          if (isValid) {
            setApiKey(storedKey)
            setAuthMethod('key')
            setIsAuthenticated(true)
            return
          }
          // Invalid key, remove it
          sessionStorage.removeItem(AUTH_STORAGE_KEY)
        }

        if (whoami?.authenticated && whoami.user) {
          setUser(whoami.user)
          setAuthMethod('session')
          setIsAuthenticated(true)
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
  }, [validateApiKey])

  // A 401 from any API call means the key or session died mid-flight; drop
  // back to the login dialog
  useEffect(() => {
    const onUnauthorized = () => {
      sessionStorage.removeItem(AUTH_STORAGE_KEY)
      setApiKey(null)
      setUser(null)
      setAuthMethod(null)
      setIsAuthenticated(false)
    }
    window.addEventListener('llamactl:unauthorized', onUnauthorized)
    return () => window.removeEventListener('llamactl:unauthorized', onUnauthorized)
  }, [])

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
      setAuthMethod('key')
      setIsAuthenticated(true)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Authentication failed'
      setError(errorMessage)
      throw new Error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }, [validateApiKey])

  const logout = useCallback(() => {
    if (authMethod === 'session') {
      // Revoke the server-side session; best-effort, the local state is
      // cleared regardless
      void fetch(`${document.baseURI}api/v1/auth/oidc/logout`, {
        method: 'POST',
        keepalive: true,
      }).catch(() => {})
    }
    sessionStorage.removeItem(AUTH_STORAGE_KEY)
    setApiKey(null)
    setUser(null)
    setAuthMethod(null)
    setIsAuthenticated(false)
    setError(null)
  }, [authMethod])

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
  }, [apiKey, logout, validateApiKey])

  const value: AuthContextType = {
    isAuthenticated,
    isLoading,
    apiKey,
    user,
    authMethod,
    oidcEnabled,
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
