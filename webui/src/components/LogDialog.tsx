import React, { useState, useEffect, useRef } from 'react'
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
import { Badge } from '@/components/ui/badge'
import { instancesApi } from '@/lib/api'
import { 
  RefreshCw, 
  Download, 
  Copy, 
  CheckCircle, 
  AlertCircle,
  Loader2,
  Settings
} from 'lucide-react'

interface LogsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instanceName: string
  isRunning: boolean
}

const LogsDialog: React.FC<LogsDialogProps> = ({
  open,
  onOpenChange,
  instanceName,
  isRunning
}) => {
  const [logs, setLogs] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lineCount, setLineCount] = useState(100)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  
  const logContainerRef = useRef<HTMLDivElement>(null)
  const refreshIntervalRef = useRef<NodeJS.Timeout | null>(null)

  // Fetch logs function
  const fetchLogs = React.useCallback(
    async (lines?: number) => {
      if (!instanceName) return
      
      setLoading(true)
      setError(null)
      
      try {
        const logText = await instancesApi.getLogs(instanceName, lines)
        setLogs(logText)
        
        // Auto-scroll to bottom
        setTimeout(() => {
          if (logContainerRef.current) {
            logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
          }
        }, 100)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch logs')
      } finally {
        setLoading(false)
      }
    },
    [instanceName]
  )

  // Initial load when dialog opens
  useEffect(() => {
    if (open && instanceName) {
      void fetchLogs(lineCount)
    }
  }, [open, instanceName, fetchLogs, lineCount])

  // Auto-refresh effect
  useEffect(() => {
    if (autoRefresh && isRunning && open) {
      refreshIntervalRef.current = setInterval(() => {
        void fetchLogs(lineCount)
      }, 2000) // Refresh every 2 seconds
    } else {
      if (refreshIntervalRef.current) {
        clearInterval(refreshIntervalRef.current)
        refreshIntervalRef.current = null
      }
    }

    return () => {
      if (refreshIntervalRef.current) {
        clearInterval(refreshIntervalRef.current)
      }
    }
  }, [autoRefresh, isRunning, open, lineCount, fetchLogs])

  // Copy logs to clipboard
  const copyLogs = async () => {
    try {
      await navigator.clipboard.writeText(logs)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy logs:', err)
    }
  }

  // Download logs as file
  const downloadLogs = () => {
    const blob = new Blob([logs], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${instanceName}-logs.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  // Handle line count change
  const handleLineCountChange = (value: string) => {
    const num = parseInt(value) || 100
    setLineCount(num)
  }

  // Apply new line count
  const applyLineCount = () => {
    void fetchLogs(lineCount)
    setShowSettings(false)
  }

  // Format logs with basic syntax highlighting
  const formatLogs = (logText: string) => {
    if (!logText) return ''
    
    return logText.split('\n').map((line, index) => {
      let className = 'font-mono text-sm leading-relaxed'
      
      // Basic log level detection
      if (line.includes('ERROR') || line.includes('[ERROR]')) {
        className += ' text-red-400'
      } else if (line.includes('WARN') || line.includes('[WARN]')) {
        className += ' text-yellow-400'
      } else if (line.includes('INFO') || line.includes('[INFO]')) {
        className += ' text-blue-400'
      } else if (line.includes('DEBUG') || line.includes('[DEBUG]')) {
        className += ' text-gray-400'
      } else if (line.includes('===')) {
        className += ' text-green-400 font-semibold'
      } else {
        className += ' text-gray-300'
      }
      
      return (
        <div key={index} className={className}>
          {line || '\u00A0'} {/* Non-breaking space for empty lines */}
        </div>
      )
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-w-[calc(100%-2rem)] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Logs: {instanceName}
                <Badge variant={isRunning ? "default" : "secondary"}>
                  {isRunning ? "Running" : "Stopped"}
                </Badge>
              </DialogTitle>
              <DialogDescription>
                Instance logs and output
              </DialogDescription>
            </div>
            
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowSettings(!showSettings)}
              >
                <Settings className="h-4 w-4" />
              </Button>
              
              <Button
                variant="outline"
                size="sm"
                onClick={() => void fetchLogs(lineCount)}
                disabled={loading}
              >
                {loading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        </DialogHeader>

        {/* Settings Panel */}
        {showSettings && (
          <div className="border rounded-lg p-4 bg-muted/50">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Label htmlFor="lineCount">Lines:</Label>
                <Input
                  id="lineCount"
                  type="number"
                  value={lineCount}
                  onChange={(e) => handleLineCountChange(e.target.value)}
                  className="w-24"
                  min="1"
                  max="10000"
                />
              </div>
              
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="autoRefresh"
                  checked={autoRefresh}
                  onChange={(e) => setAutoRefresh(e.target.checked)}
                  disabled={!isRunning}
                  className="rounded"
                />
                <Label htmlFor="autoRefresh">
                  Auto-refresh {!isRunning && '(instance not running)'}
                </Label>
              </div>
              
              <Button size="sm" onClick={applyLineCount}>
                Apply
              </Button>
            </div>
          </div>
        )}

        {/* Log Content */}
        <div className="flex-1 flex flex-col min-h-0">
          {error && (
            <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg mb-4">
              <AlertCircle className="h-4 w-4 text-destructive" />
              <span className="text-sm text-destructive">{error}</span>
            </div>
          )}
          
          <div 
            ref={logContainerRef}
            className="flex-1 bg-gray-900 rounded-lg p-4 overflow-auto min-h-[400px] max-h-[500px]"
          >
            {loading && !logs ? (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
                <span className="ml-2 text-gray-400">Loading logs...</span>
              </div>
            ) : logs ? (
              <div className="whitespace-pre-wrap">
                {formatLogs(logs)}
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-400">
                No logs available
              </div>
            )}
          </div>
          
          {autoRefresh && isRunning && (
            <div className="flex items-center gap-2 mt-2 text-sm text-muted-foreground">
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              Auto-refreshing every 2 seconds
            </div>
          )}
        </div>

        <DialogFooter className="flex-shrink-0">
          <div className="flex items-center gap-2 w-full">
            <Button
              variant="outline"
              onClick={() => void copyLogs()}
              disabled={!logs}
            >
              {copied ? (
                <>
                  <CheckCircle className="h-4 w-4" />
                  Copied!
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4" />
                  Copy
                </>
              )}
            </Button>
            
            <Button
              variant="outline"
              onClick={downloadLogs}
              disabled={!logs}
            >
              <Download className="h-4 w-4" />
              Download
            </Button>
            
            <div className="flex-1" />
            
            <Button onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default LogsDialog