export interface DownloadJob {
  id: string
  repo: string
  tag: string
  status: 'queued' | 'downloading' | 'completed' | 'failed' | 'cancelled'
  progress: {
    bytes_downloaded: number
    total_bytes: number
    current_file: string
  }
  error: string | null
  created_at: number
  completed_at: number | null
}

export interface CachedModel {
  repo: string
  tag: string
  files: ModelFile[]
  size_bytes: number
}

export interface ModelFile {
  name: string
  path: string
  size_bytes: number
  type: 'gguf' | 'preset' | 'mmproj'
}

// Combined row type for the table
export type ModelRow =
  | { type: 'downloading'; job: DownloadJob }
  | { type: 'cached'; model: CachedModel }
