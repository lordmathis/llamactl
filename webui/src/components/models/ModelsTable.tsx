import { useState } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { ModelRow, DownloadJob, CachedModel, ModelFile } from '@/types/model'
import { formatBytes } from '@/lib/utils'
import { useModels } from '@/contexts/ModelsContext'
import {
  Download,
  X,
  Trash2,
  Loader2,
  ChevronDown,
  ChevronRight,
  FileCode,
  FileText,
  Image,
  RefreshCw,
} from 'lucide-react'
import StatusBadge from './StatusBadge'

interface ModelsTableProps {
  rows: ModelRow[]
}

export default function ModelsTable({ rows }: ModelsTableProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())

  const toggleRow = (key: string) => {
    setExpandedRows(prev => {
      const newSet = new Set(prev)
      if (newSet.has(key)) {
        newSet.delete(key)
      } else {
        newSet.add(key)
      }
      return newSet
    })
  }

  if (rows.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <Download className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p className="text-lg mb-2">No models yet</p>
        <p className="text-sm">Download your first model to get started</p>
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Repository</TableHead>
          <TableHead>Tag</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Size</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => {
          if (row.type === 'downloading') {
            return <DownloadingRow key={row.job.id} job={row.job} />
          } else {
            const key = `${row.model.repo}:${row.model.tag}`
            const expanded = expandedRows.has(key)
            return (
              <>
                <CachedModelRow
                  key={key}
                  model={row.model}
                  expanded={expanded}
                  onToggle={() => toggleRow(key)}
                />
                {expanded && (
                  <ExpandedFilesRow key={`${key}-files`} files={row.model.files} />
                )}
              </>
            )
          }
        })}
      </TableBody>
    </Table>
  )
}

function DownloadingRow({ job }: { job: DownloadJob }) {
  const { cancelDownload } = useModels()
  const [cancelling, setCancelling] = useState(false)

  const handleCancel = async () => {
    if (!confirm('Cancel this download? Partial files will be deleted.')) {
      return
    }

    setCancelling(true)
    try {
      await cancelDownload(job.id)
    } catch (error) {
      console.error('Failed to cancel download:', error)
    } finally {
      setCancelling(false)
    }
  }

  const handleRetry = async () => {
    const { startDownload } = useModels()
    try {
      await startDownload(job.repo, job.tag)
    } catch (error) {
      console.error('Failed to retry download:', error)
    }
  }

  // Failed job display
  if (job.status === 'failed') {
    return (
      <TableRow>
        <TableCell className="font-medium" title={job.repo}>
          {job.repo}
        </TableCell>
        <TableCell>
          <Badge variant="outline">{job.tag}</Badge>
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            <StatusBadge status="failed" />
            {job.error && (
              <span
                className="text-xs text-muted-foreground truncate max-w-xs"
                title={job.error}
              >
                {job.error}
              </span>
            )}
          </div>
        </TableCell>
        <TableCell>-</TableCell>
        <TableCell className="text-right">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleRetry}
            title="Retry download"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </TableCell>
      </TableRow>
    )
  }

  // Active download display
  const progress =
    job.progress.total_bytes > 0
      ? (job.progress.bytes_downloaded / job.progress.total_bytes) * 100
      : 0

  return (
    <TableRow>
      <TableCell className="font-medium" title={job.repo}>
        {job.repo}
      </TableCell>
      <TableCell>
        <Badge variant="outline">{job.tag}</Badge>
      </TableCell>
      <TableCell>
        <div className="space-y-1 min-w-[200px]">
          <div className="flex items-center gap-2">
            <div className="flex-1 bg-muted rounded-full h-2">
              <div
                className="bg-primary h-2 rounded-full transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground min-w-[40px] text-right">
              {progress.toFixed(1)}%
            </span>
          </div>
          {job.progress.current_file && (
            <p className="text-xs text-muted-foreground truncate" title={job.progress.current_file}>
              {job.progress.current_file}
            </p>
          )}
        </div>
      </TableCell>
      <TableCell>{formatBytes(job.progress.total_bytes)}</TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="icon"
          onClick={handleCancel}
          disabled={cancelling}
          title="Cancel download"
        >
          {cancelling ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <X className="h-4 w-4" />
          )}
        </Button>
      </TableCell>
    </TableRow>
  )
}

function CachedModelRow({
  model,
  expanded,
  onToggle,
}: {
  model: CachedModel
  expanded: boolean
  onToggle: () => void
}) {
  const { deleteModel } = useModels()
  const [deleting, setDeleting] = useState(false)

  const handleDelete = async () => {
    const message = model.tag
      ? `Delete ${model.repo}:${model.tag}?\n\nAll files for this quantization will be removed.`
      : `Delete ALL quantizations for ${model.repo}?\n\nThis will remove all cached files for this repository.`

    if (!confirm(message)) {
      return
    }

    setDeleting(true)
    try {
      await deleteModel(model.repo, model.tag)
    } catch (error) {
      console.error('Failed to delete model:', error)
    } finally {
      setDeleting(false)
    }
  }

  return (
    <TableRow className="cursor-pointer" onClick={onToggle}>
      <TableCell className="font-medium" title={model.repo}>
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
          {model.repo}
        </div>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{model.tag}</Badge>
      </TableCell>
      <TableCell>
        <Badge variant="default" className="bg-green-600">
          Cached
        </Badge>
      </TableCell>
      <TableCell>{formatBytes(model.size_bytes)}</TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="icon"
          onClick={(e) => {
            e.stopPropagation()
            void handleDelete()
          }}
          disabled={deleting}
          title="Delete model"
        >
          {deleting ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Trash2 className="h-4 w-4 text-destructive" />
          )}
        </Button>
      </TableCell>
    </TableRow>
  )
}

function ExpandedFilesRow({ files }: { files: ModelFile[] }) {
  const getFileIcon = (type: ModelFile['type']) => {
    switch (type) {
      case 'gguf':
        return <FileCode className="h-4 w-4 text-blue-500" />
      case 'preset':
        return <FileText className="h-4 w-4 text-yellow-500" />
      case 'mmproj':
        return <Image className="h-4 w-4 text-purple-500" />
    }
  }

  return (
    <TableRow className="bg-muted/30 hover:bg-muted/30">
      <TableCell colSpan={5} className="p-4">
        <div className="space-y-2 pl-6">
          <p className="text-sm font-semibold">Files:</p>
          <ul className="text-sm space-y-1">
            {files.map((file) => (
              <li key={file.path} className="flex items-center gap-2">
                {getFileIcon(file.type)}
                <span className="font-mono">{file.name}</span>
                <span className="text-muted-foreground">
                  ({formatBytes(file.size_bytes)})
                </span>
              </li>
            ))}
          </ul>
        </div>
      </TableCell>
    </TableRow>
  )
}
