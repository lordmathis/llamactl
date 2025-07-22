import { useState } from 'react'
import Header from '@/components/Header'
import InstanceList from '@/components/InstanceList'
import InstanceModal from '@/components/InstanceModal'
import { CreateInstanceOptions, Instance } from '@/types/instance'
import { useInstances } from '@/contexts/InstancesContext'

function App() {
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingInstance, setEditingInstance] = useState<Instance | undefined>(undefined)
  const { createInstance, updateInstance } = useInstances()

  const handleCreateInstance = () => {
    setEditingInstance(undefined)
    setIsModalOpen(true)
  }

  const handleEditInstance = (instance: Instance) => {
    setEditingInstance(instance)
    setIsModalOpen(true)
  }

  const handleSaveInstance = (name: string, options: CreateInstanceOptions) => {
    if (editingInstance) {
      updateInstance(editingInstance.name, options)
    } else {
      createInstance(name, options)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Header onCreateInstance={handleCreateInstance} />
      <main className="container mx-auto max-w-4xl px-4 py-8">
        <InstanceList editInstance={handleEditInstance} />
      </main>
      
      <InstanceModal
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        onSave={handleSaveInstance}
        instance={editingInstance}
      />
    </div>
  )
}

export default App