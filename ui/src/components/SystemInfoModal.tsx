// ui/src/components/SystemInfoModal.tsx
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
import { 
  RefreshCw, 
  AlertCircle,
  Loader2,
  ChevronDown,
  ChevronRight,
  Monitor,
  HelpCircle
} from 'lucide-react'
import { serverApi } from '@/lib/api'

interface SystemInfoModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface SystemInfo {
  version: string
  devices: string
  help: string
}

const SystemInfoModal: React.FC<SystemInfoModalProps> = ({
  open,
  onOpenChange
}) => {
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showHelp, setShowHelp] = useState(false)

  // Fetch system info
  const fetchSystemInfo = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const [version, devices, help] = await Promise.all([
        serverApi.getVersion(),
        serverApi.getDevices(),
        serverApi.getHelp()
      ])
      
      setSystemInfo({ version, devices, help })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch system info')
    } finally {
      setLoading(false)
    }
  }

  // Load data when modal opens
  useEffect(() => {
    if (open) {
      fetchSystemInfo()
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange} >
      <DialogContent className="sm:max-w-4xl max-w-[calc(100%-2rem)] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <Monitor className="h-5 w-5" />
                System Information
              </DialogTitle>
              <DialogDescription>
                Llama.cpp server environment and capabilities
              </DialogDescription>
            </div>
            
            <Button
              variant="outline"
              size="sm"
              onClick={fetchSystemInfo}
              disabled={loading}
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
            </Button>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          {loading && !systemInfo ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
              <span className="ml-2 text-gray-400">Loading system information...</span>
            </div>
          ) : error ? (
            <div className="flex items-center gap-2 p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
              <AlertCircle className="h-4 w-4 text-destructive" />
              <span className="text-sm text-destructive">{error}</span>
            </div>
          ) : systemInfo ? (
            <div className="space-y-6">
              {/* Version Section */}
              <div className="space-y-3">
                <h3 className="font-semibold">Version</h3>
                
                <div className="bg-gray-900 rounded-lg p-4">
                  <div className="mb-2">
                    <span className="text-sm text-gray-400">$ llama-server --version</span>
                  </div>
                  <pre className="text-sm text-gray-300 whitespace-pre-wrap font-mono">
                    {systemInfo.version}
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
                    {systemInfo.devices}
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
                      {systemInfo.help}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          ) : null}
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

export default SystemInfoModal