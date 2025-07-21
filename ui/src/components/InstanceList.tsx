import { useInstances } from '@/hooks/useInstances'
import InstanceCard from '@/components/InstanceCard'

function InstanceList() {
  const { instances, loading, error } = useInstances()

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading instances...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-red-600 mb-4">
          <p className="text-lg font-semibold">Error loading instances</p>
          <p className="text-sm">{error}</p>
        </div>
      </div>
    )
  }

  if (instances.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-600 text-lg mb-2">No instances found</p>
        <p className="text-gray-500 text-sm">Create your first instance to get started</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-gray-900 mb-6">
        Instances ({instances.length})
      </h2>
      
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {instances.map((instance) => (
          <InstanceCard 
            key={instance.name} 
            instance={instance}
          />
        ))}
      </div>
    </div>
  )
}

export default InstanceList