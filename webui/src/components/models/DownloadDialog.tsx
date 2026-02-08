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
  const [repo, setRepo] = useState('')
  const [tag, setTag] = useState('')
  const [repoError, setRepoError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const { startDownload } = useModels()

  const validateRepo = (value: string): boolean => {
    if (!value) {
      setRepoError('Repository is required')
      return false
    }
    if (!value.includes('/')) {
      setRepoError('Format must be: org/model-name')
      return false
    }
    setRepoError('')
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateRepo(repo)) return

    setIsSubmitting(true)
    try {
      await startDownload(repo, tag || undefined)
      onOpenChange(false)
      // Reset form
      setRepo('')
      setTag('')
      setRepoError('')
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
    setRepo('')
    setTag('')
    setRepoError('')
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
              <Label htmlFor="repo">Repository *</Label>
              <Input
                id="repo"
                value={repo}
                onChange={(e) => {
                  setRepo(e.target.value)
                  if (repoError) validateRepo(e.target.value)
                }}
                onBlur={(e) => validateRepo(e.target.value)}
                placeholder="bartowski/Llama-3.2-3B-Instruct-GGUF"
                className={repoError ? 'border-red-500' : ''}
                disabled={isSubmitting}
              />
              {repoError && (
                <p className="text-sm text-red-500 mt-1">{repoError}</p>
              )}
            </div>

            <div>
              <Label htmlFor="tag">Tag (optional)</Label>
              <Input
                id="tag"
                value={tag}
                onChange={(e) => setTag(e.target.value)}
                placeholder="Q4_K_M (leave empty for 'latest')"
                disabled={isSubmitting}
              />
              <p className="text-sm text-muted-foreground mt-1">
                Quantization or variant name
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
            <Button type="submit" disabled={!repo || !!repoError || isSubmitting}>
              {isSubmitting ? 'Starting...' : 'Start Download'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
