import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Download } from 'lucide-react'
import { useModels } from '@/contexts/ModelsContext'
import DownloadDialog from './DownloadDialog'

export default function ModelsHeader() {
  const { models, activeJobs } = useModels()
  const [downloadDialogOpen, setDownloadDialogOpen] = useState(false)

  const totalCount = (models?.length || 0) + (activeJobs?.length || 0)

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Models ({totalCount})</h2>
        <Button onClick={() => setDownloadDialogOpen(true)}>
          <Download className="h-4 w-4 mr-2" />
          Download Model
        </Button>
      </div>

      <DownloadDialog
        open={downloadDialogOpen}
        onOpenChange={setDownloadDialogOpen}
      />
    </>
  )
}
