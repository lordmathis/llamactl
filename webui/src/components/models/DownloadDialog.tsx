import { useState } from 'react'
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
import { useModels } from '@/contexts/ModelsContext'

interface DownloadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function DownloadDialog({ open, onOpenChange }: DownloadDialogProps) {
  const [model, setModel] = useState('')
  const [modelError, setModelError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const { startDownload } = useModels()

  const validateModel = (value: string): boolean => {
    if (!value) {
      setModelError('Model is required')
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
      // Parse repo and tag from format "org/model:tag"
      const colonIdx = model.lastIndexOf(':')
      const repo = colonIdx !== -1 ? model.substring(0, colonIdx) : model
      const tag = colonIdx !== -1 ? model.substring(colonIdx + 1) : undefined

      await startDownload(repo, tag)
      onOpenChange(false)
      // Reset form
      setModel('')
      setModelError('')
    } catch (error) {
      // Error is handled by context
      console.error('Failed to start download:', error)
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleCancel = () => {
    onOpenChange(false)
    // Reset form
    setModel('')
    setModelError('')
  }

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
                placeholder="bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M"
                className={modelError ? 'border-red-500' : ''}
                disabled={isSubmitting}
              />
              {modelError && (
                <p className="text-sm text-red-500 mt-1">{modelError}</p>
              )}
              <p className="text-sm text-muted-foreground mt-1">
                Format: org/model-name or org/model-name:tag (leave tag empty for latest)
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
