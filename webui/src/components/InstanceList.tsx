// ui/src/components/InstanceList.tsx
import { useInstances } from '@/contexts/InstancesContext'
import InstanceCard from '@/components/InstanceCard'
import type { Instance } from '@/types/instance'
import { memo } from 'react'

interface InstanceListProps {
  editInstance: (instance: Instance) => void
}

// Memoize InstanceCard to prevent re-renders when other instances change
const MemoizedInstanceCard = memo(InstanceCard)

function InstanceList({ editInstance }: InstanceListProps) {
  const { instances, loading, error, startInstance, stopInstance, deleteInstance } = useInstances()

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12" aria-label="Loading">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading instances...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-destructive mb-4">
          <p className="text-lg font-semibold">Error loading instances</p>
          <p className="text-sm">{error}</p>
        </div>
      </div>
    )
  }

  if (instances.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-foreground text-lg mb-2">No instances found</p>
        <p className="text-muted-foreground text-sm">Create your first instance to get started</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-foreground mb-6">
        Instances ({instances.length})
      </h2>
      
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {instances.map((instance) => (
          <MemoizedInstanceCard 
            key={instance.name} 
            instance={instance}
            startInstance={() => { void startInstance(instance.name) }}
            stopInstance={() => { void stopInstance(instance.name) }}
            deleteInstance={() => { void deleteInstance(instance.name) }}
            editInstance={editInstance}
          />
        ))}
      </div>
    </div>
  )
}

export default InstanceList