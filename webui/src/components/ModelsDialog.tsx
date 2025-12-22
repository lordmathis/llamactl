import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { llamaCppApi } from '@/lib/api'
import { RefreshCw, Loader2, AlertCircle } from 'lucide-react'

interface ModelsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instanceName: string
  isRunning: boolean
}

interface Model {
  id: string
  object: string
  owned_by: string
  created: number
  in_cache: boolean
  path: string
  status: {
    value: string // "loaded" | "loading" | "unloaded"
    args: string[]
  }
}

const StatusIcon: React.FC<{ status: string }> = ({ status }) => {
  switch (status) {
    case 'loaded':
      return (
        <div className="h-2 w-2 rounded-full bg-green-500" />
      )
    case 'loading':
      return (
        <Loader2
          className="h-3 w-3 animate-spin text-yellow-500"
        />
      )
    case 'unloaded':
      return (
        <div className="h-2 w-2 rounded-full bg-gray-400" />
      )
    default:
      return null
  }
}

const ModelsDialog: React.FC<ModelsDialogProps> = ({
  open,
  onOpenChange,
  instanceName,
  isRunning,
}) => {
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loadingModels, setLoadingModels] = useState<Set<string>>(new Set())

  // Fetch models function
  const fetchModels = React.useCallback(async () => {
    if (!instanceName || !isRunning) return

    setLoading(true)
    setError(null)

    try {
      const response = await llamaCppApi.getModels(instanceName)
      setModels(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch models')
    } finally {
      setLoading(false)
    }
  }, [instanceName, isRunning])

  // Fetch models when dialog opens
  useEffect(() => {
    if (!open || !isRunning) return

    // Initial fetch
    void fetchModels()
  }, [open, isRunning, fetchModels])

  // Auto-refresh only when models are loading
  useEffect(() => {
    if (!open || !isRunning) return

    // Check if any model is in loading state
    const hasLoadingModel = models.some(m => m.status.value === 'loading')

    if (!hasLoadingModel) return

    // Poll every 2 seconds when there's a loading model
    const interval = setInterval(() => {
      void fetchModels()
    }, 2000)

    return () => clearInterval(interval)
  }, [open, isRunning, models, fetchModels])

  // Load model
  const loadModel = async (modelName: string) => {
    setLoadingModels((prev) => new Set(prev).add(modelName))
    setError(null)

    try {
      await llamaCppApi.loadModel(instanceName, modelName)
      // Wait a bit for the backend to process the load
      await new Promise(resolve => setTimeout(resolve, 500))
      // Refresh models list after loading
      await fetchModels()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load model')
    } finally {
      setLoadingModels((prev) => {
        const newSet = new Set(prev)
        newSet.delete(modelName)
        return newSet
      })
    }
  }

  // Unload model
  const unloadModel = async (modelName: string) => {
    setLoadingModels((prev) => new Set(prev).add(modelName))
    setError(null)

    try {
      await llamaCppApi.unloadModel(instanceName, modelName)
      // Wait a bit for the backend to process the unload
      await new Promise(resolve => setTimeout(resolve, 500))
      // Refresh models list after unloading
      await fetchModels()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unload model')
    } finally {
      setLoadingModels((prev) => {
        const newSet = new Set(prev)
        newSet.delete(modelName)
        return newSet
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-w-[calc(100%-2rem)] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Models: {instanceName}
                <Badge variant={isRunning ? 'default' : 'secondary'}>
                  {isRunning ? 'Running' : 'Stopped'}
                </Badge>
              </DialogTitle>
              <DialogDescription>
                Manage models in this llama.cpp instance
              </DialogDescription>
            </div>

            <Button
              variant="outline"
              size="sm"
              onClick={() => void fetchModels()}
              disabled={loading || !isRunning}
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
            </Button>
          </div>
        </DialogHeader>

        {/* Error Display */}
        {error && (
          <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <span className="text-sm text-destructive">{error}</span>
          </div>
        )}

        {/* Models Table */}
        <div className="flex-1 flex flex-col min-h-0 overflow-auto">
          {!isRunning ? (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              Instance is not running
            </div>
          ) : loading && models.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">
                Loading models...
              </span>
            </div>
          ) : models.length === 0 ? (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              No models found
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Model</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.map((model) => {
                  const isLoading = loadingModels.has(model.id)
                  const isModelLoading = model.status.value === 'loading'

                  return (
                    <TableRow key={model.id}>
                      <TableCell className="font-mono text-sm">
                        {model.id}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <StatusIcon status={model.status.value} />
                          <span className="text-sm capitalize">
                            {model.status.value}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        {model.status.value === 'loaded' ? (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => { void unloadModel(model.id) }}
                            disabled={!isRunning || isLoading || isModelLoading}
                          >
                            {isLoading ? (
                              <>
                                <Loader2 className="h-3 w-3 animate-spin mr-1" />
                                Unloading...
                              </>
                            ) : (
                              'Unload'
                            )}
                          </Button>
                        ) : model.status.value === 'unloaded' ? (
                          <Button
                            size="sm"
                            variant="default"
                            onClick={() => { void loadModel(model.id) }}
                            disabled={!isRunning || isLoading || isModelLoading}
                          >
                            {isLoading ? (
                              <>
                                <Loader2 className="h-3 w-3 animate-spin mr-1" />
                                Loading...
                              </>
                            ) : (
                              'Load'
                            )}
                          </Button>
                        ) : (
                          <Button size="sm" variant="ghost" disabled>
                            Loading...
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </div>

        {/* Auto-refresh indicator - only shown when models are loading */}
        {isRunning && models.some(m => m.status.value === 'loading') && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse"></div>
            Auto-refreshing while models are loading
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

export default ModelsDialog
