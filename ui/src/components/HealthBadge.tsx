// ui/src/components/HealthBadge.tsx
import React from 'react'
import { Badge } from '@/components/ui/badge'
import { HealthStatus } from '@/types/instance'
import { CheckCircle, Loader2, XCircle } from 'lucide-react'

interface HealthBadgeProps {
  health?: HealthStatus
}

const HealthBadge: React.FC<HealthBadgeProps> = ({ health }) => {
  if (!health) return null

  const getIcon = () => {
    switch (health.status) {
      case 'ok': return <CheckCircle className="h-3 w-3" />
      case 'loading': return <Loader2 className="h-3 w-3 animate-spin" />
      case 'error': return <XCircle className="h-3 w-3" />
    }
  }

  const getVariant = () => {
    switch (health.status) {
      case 'ok': return 'default'
      case 'loading': return 'outline'
      case 'error': return 'destructive'
    }
  }

  const getText = () => {
    switch (health.status) {
      case 'ok': return 'Ready'
      case 'loading': return 'Loading'
      case 'error': return 'Error'
    }
  }

  return (
    <Badge 
      variant={getVariant()} 
      className={`flex items-center gap-1.5 ${health.status === 'ok' ? 'bg-green-100 text-green-800 border-green-200 dark:bg-green-900 dark:text-green-200 dark:border-green-800' : ''}`}
    >
      {getIcon()}
      <span className="text-xs">{getText()}</span>
    </Badge>
  )
}

export default HealthBadge