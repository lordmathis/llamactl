import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import SelectInput from '@/components/form/SelectInput'
import {
  RefreshCw,
  AlertCircle,
  Loader2,
  ChevronDown,
  ChevronRight,
  Monitor,
  HelpCircle,
  Info
} from 'lucide-react'
import { serverApi } from '@/lib/api'

// Helper to get version from environment
const getAppVersion = (): string => {
  try {
    return (import.meta.env as Record<string, string>).VITE_APP_VERSION || 'unknown'
  } catch {
    return 'unknown'
  }
}

interface BackendInfoDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface BackendInfo {
  version: string
  devices: string
  help: string
}

type BackendType = 'llama-cpp' | 'mlx' | 'vllm'

const BACKEND_OPTIONS = [
  { value: 'llama-cpp', label: 'Llama.cpp' },
  { value: 'mlx', label: 'MLX' },
  { value: 'vllm', label: 'vLLM' },
] as const

const BackendInfoDialog: React.FC<BackendInfoDialogProps> = ({
  open,
  onOpenChange
}) => {
  const [selectedBackend, setSelectedBackend] = useState<BackendType>('llama-cpp')
  const [backendInfo, setBackendInfo] = useState<BackendInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showHelp, setShowHelp] = useState(false)

  // Fetch backend info
  const fetchBackendInfo = async (backend: BackendType) => {
    if (backend !== 'llama-cpp') {
      setBackendInfo(null)
      setError(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      const [version, devices, help] = await Promise.all([
        serverApi.getVersion(),
        serverApi.getDevices(),
        serverApi.getHelp()
      ])

      setBackendInfo({ version, devices, help })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch backend info')
    } finally {
      setLoading(false)
    }
  }

  // Load data when dialog opens or backend changes
  useEffect(() => {
    if (open) {
      fetchBackendInfo(selectedBackend)
    }
  }, [open, selectedBackend])

  const handleBackendChange = (value: string) => {
    setSelectedBackend(value as BackendType)
    setShowHelp(false) // Reset help section when switching backends
  }

  const renderBackendContent = () => {
    if (selectedBackend !== 'llama-cpp') {
      return (
        <div className="flex items-center justify-center py-12">
          <div className="text-center space-y-3">
            <Info className="h-12 w-12 text-gray-400 mx-auto" />
            <div>
              <h3 className="font-semibold text-gray-700">Backend Info Not Available</h3>
              <p className="text-sm text-gray-500 mt-1">
                Information for {BACKEND_OPTIONS.find(b => b.value === selectedBackend)?.label} backend is not yet implemented.
              </p>
            </div>
          </div>
        </div>
      )
    }

    if (loading && !backendInfo) {
      return (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-400">Loading backend information...</span>
        </div>
      )
    }

    if (error) {
      return (
        <div className="flex items-center gap-2 p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
          <AlertCircle className="h-4 w-4 text-destructive" />
          <span className="text-sm text-destructive">{error}</span>
        </div>
      )
    }

    if (!backendInfo) {
      return null
    }

    return (
      <div className="space-y-6">
        {/* Llamactl Version Section */}
        <div className="space-y-3">
          <h3 className="font-semibold">Llamactl Version</h3>

          <div className="bg-gray-900 rounded-lg p-4">
            <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono">
              {getAppVersion()}
            </pre>
          </div>
        </div>

        {/* Backend Version Section */}
        <div className="space-y-3">
          <h3 className="font-semibold">
            {BACKEND_OPTIONS.find(b => b.value === selectedBackend)?.label} Version
          </h3>

          <div className="bg-gray-900 rounded-lg p-4">
            <div className="mb-2">
              <span className="text-sm text-gray-400">$ llama-server --version</span>
            </div>
            <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono">
              {backendInfo.version}
            </pre>
          </div>
        </div>

        {/* Devices Section */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold">Available Devices</h3>
          </div>

          <div className="bg-gray-900 rounded-lg p-4">
            <div className="mb-2">
              <span className="text-sm text-gray-400">$ llama-server --list-devices</span>
            </div>
            <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono">
              {backendInfo.devices}
            </pre>
          </div>
        </div>

        {/* Help Section */}
        <div className="space-y-3">
          <Button
            variant="ghost"
            onClick={() => setShowHelp(!showHelp)}
            className="flex items-center gap-2 p-0 h-auto font-semibold"
          >
            {showHelp ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <HelpCircle className="h-4 w-4" />
            Command Line Options
          </Button>

          {showHelp && (
            <div className="bg-gray-900 rounded-lg p-4">
              <div className="mb-2">
                <span className="text-sm text-gray-400">$ llama-server --help</span>
              </div>
              <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono max-h-64 overflow-y-auto">
                {backendInfo.help}
              </pre>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-w-[calc(100%-2rem)] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <Monitor className="h-5 w-5" />
                Backend Information
              </DialogTitle>
              <DialogDescription>
                View backend-specific environment and capabilities
              </DialogDescription>
            </div>

            <div className="flex items-center gap-2">
              <div className="w-32">
                <SelectInput
                  id="backend-select"
                  label=""
                  value={selectedBackend}
                  onChange={(value) => handleBackendChange(value || 'llama-cpp')}
                  options={BACKEND_OPTIONS}
                  className="text-sm"
                />
              </div>

              {selectedBackend === 'llama-cpp' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => fetchBackendInfo(selectedBackend)}
                  disabled={loading}
                >
                  {loading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4" />
                  )}
                </Button>
              )}
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          {renderBackendContent()}
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default BackendInfoDialog