import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { AlertCircle, Key, Loader2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'

interface LoginDialogProps {
  open: boolean
  onOpenChange?: (open: boolean) => void
}

const LoginDialog: React.FC<LoginDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const { login, isLoading, error, clearError } = useAuth()
  const [apiKey, setApiKey] = useState('')
  const [localLoading, setLocalLoading] = useState(false)

  // Clear form and errors when dialog opens/closes
  useEffect(() => {
    if (open) {
      setApiKey('')
      clearError()
    }
  }, [open, clearError])

  // Clear error when user starts typing
  useEffect(() => {
    if (error && apiKey) {
      clearError()
    }
  }, [apiKey, error, clearError])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!apiKey.trim()) {
      return
    }

    setLocalLoading(true)
    
    try {
      await login(apiKey.trim())
      // Login successful - dialog will close automatically when auth state changes
      setApiKey('')
    } catch (err) {
      // Error is handled by the AuthContext
      console.error('Login failed:', err)
    } finally {
      setLocalLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !isSubmitDisabled) {
      // Create a synthetic FormEvent to satisfy handleSubmit's type
      const syntheticEvent = {
        preventDefault: () => {},
      } as React.FormEvent<HTMLFormElement>;
      void handleSubmit(syntheticEvent)
    }
  }

  const isSubmitDisabled = !apiKey.trim() || isLoading || localLoading

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent 
        className="sm:max-w-md" 
        showCloseButton={false} // Prevent closing without auth
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Authentication Required
          </DialogTitle>
          <DialogDescription>
            Please enter your management API key to access the Llamactl dashboard.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={(e) => { void handleSubmit(e) }}>
          <div className="grid gap-4 py-4">
            {/* Error Display */}
            {error && (
              <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
                <AlertCircle className="h-4 w-4 text-destructive flex-shrink-0" />
                <span className="text-sm text-destructive">{error}</span>
              </div>
            )}

            {/* API Key Input */}
            <div className="grid gap-2">
              <Label htmlFor="apiKey">
                Management API Key <span className="text-red-500">*</span>
              </Label>
              <Input
                id="apiKey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="sk-management-..."
                disabled={isLoading || localLoading}
                className={error ? "border-red-500" : ""}
                autoFocus
                autoComplete="off"
              />
              <p className="text-sm text-muted-foreground">
                Your management API key is required to access instance management features.
              </p>
            </div>
          </div>

          <DialogFooter className="flex gap-2">
            <Button
              type="submit"
              disabled={isSubmitDisabled}
              data-testid="login-submit-button"
            >
              {(isLoading || localLoading) ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Authenticating...
                </>
              ) : (
                <>
                  <Key className="h-4 w-4" />
                  Login
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default LoginDialog