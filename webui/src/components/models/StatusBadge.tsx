import { Badge } from "@/components/ui/badge"
import type { DownloadJob } from "@/types/model"

interface StatusBadgeProps {
  status: DownloadJob['status']
}

export default function StatusBadge({ status }: StatusBadgeProps) {
  const variants = {
    queued: { variant: 'secondary' as const, label: 'Queued' },
    downloading: { variant: 'default' as const, label: 'Downloading' },
    completed: { variant: 'default' as const, label: 'Completed' },
    failed: { variant: 'destructive' as const, label: 'Failed' },
    cancelled: { variant: 'outline' as const, label: 'Cancelled' },
  }

  const { variant, label } = variants[status]

  return <Badge variant={variant}>{label}</Badge>
}
