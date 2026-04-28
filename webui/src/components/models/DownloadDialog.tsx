import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import SelectInput from '@/components/form/SelectInput'
import { useModels } from '@/contexts/ModelsContext'
import { nodesApi, type NodesMap } from '@/lib/api'
import { ModelFormat } from '@/types/model'

interface DownloadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const formatOptions = [
  { value: ModelFormat.GGUF, label: 'GGUF' },
  { value: ModelFormat.SAFETENSORS, label: 'Safetensors' },
]

export default function DownloadDialog({ open, onOpenChange }: DownloadDialogProps) {
  const [model, setModel] = useState('')
  const [modelError, setModelError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [nodes, setNodes] = useState<NodesMap>({})
  const [loadingNodes, setLoadingNodes] = useState(true)
  const [selectedNode, setSelectedNode] = useState<string | undefined>(undefined)
  const [format, setFormat] = useState<ModelFormat>(ModelFormat.GGUF)
  const { startDownload } = useModels()

  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const fetchedNodes = await nodesApi.list()
        setNodes(fetchedNodes)

        const nodeNames = Object.keys(fetchedNodes)
        if (nodeNames.length > 0) {
          if (!selectedNode || !nodeNames.includes(selectedNode)) {
            setSelectedNode(nodeNames[0])
          }
        } else if (selectedNode) {
          setSelectedNode(undefined)
        }
      } catch (error) {
        console.error('Failed to fetch nodes:', error)
      } finally {
        setLoadingNodes(false)
      }
    }

    void fetchNodes()
  }, [open, selectedNode])

  const validateModel = (value: string): boolean => {
    if (!value) {
      setModelError('Model is required')
      return false
    }
    if (format === ModelFormat.GGUF && !value.includes('/')) {
      setModelError('Format must be: org/model-name or org/model-name:tag')
      return false
    }
    if (!value.includes('/')) {
      setModelError('Format must be: org/model-name or org/model-name:tag')
      return false
    }
    setModelError('')
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateModel(model)) return

    setIsSubmitting(true)
    try {
      const colonIdx = model.lastIndexOf(':')
      const repo = colonIdx !== -1 ? model.substring(0, colonIdx) : model
      const tag = colonIdx !== -1 ? model.substring(colonIdx + 1) : undefined

      await startDownload(repo, tag, selectedNode, format)
      onOpenChange(false)
      setModel('')
      setModelError('')
    } catch (error) {
      console.error('Failed to start download:', error)
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleCancel = () => {
    onOpenChange(false)
    setModel('')
    setModelError('')
  }

  const nodeOptions = Object.keys(nodes).map(nodeName => ({
    value: nodeName,
    label: nodeName
  }))

  const helpText = format === ModelFormat.GGUF
    ? 'Format: org/model:quant (e.g. Q4_K_M)'
    : 'Format: org/model. Downloads all weights and config files.'

  const placeholder = format === ModelFormat.GGUF
    ? 'bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M'
    : 'meta-llama/Llama-3.2-3B'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Download Model</DialogTitle>
            <DialogDescription>
              Download a model from HuggingFace to your local cache
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {!loadingNodes && Object.keys(nodes).length > 0 && (
              <SelectInput
                id="node"
                label="Node"
                value={selectedNode}
                onChange={setSelectedNode}
                options={nodeOptions}
                description="Select the node where the model will be downloaded"
                disabled={isSubmitting}
              />
            )}

            <SelectInput
              id="format"
              label="Format"
              value={format}
              onChange={(v) => {
                setFormat((v as ModelFormat) || ModelFormat.GGUF)
                if (modelError) validateModel(model)
              }}
              options={formatOptions}
              description="Select the model format to download"
              disabled={isSubmitting}
            />

            <div>
              <Label htmlFor="model">Model *</Label>
              <Input
                id="model"
                value={model}
                onChange={(e) => {
                  setModel(e.target.value)
                  if (modelError) validateModel(e.target.value)
                }}
                onBlur={(e) => validateModel(e.target.value)}
                placeholder={placeholder}
                className={modelError ? 'border-red-500' : ''}
                disabled={isSubmitting}
              />
              {modelError && (
                <p className="text-sm text-red-500 mt-1">{modelError}</p>
              )}
              <p className="text-sm text-muted-foreground mt-1">
                {helpText}
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleCancel}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!model || !!modelError || isSubmitting}>
              {isSubmitting ? 'Starting...' : 'Start Download'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
