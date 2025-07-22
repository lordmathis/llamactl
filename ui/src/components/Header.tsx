import { useState } from 'react'
import { Button } from '@/components/ui/button'
import InstanceModal from '@/components/InstanceModal'
import { CreateInstanceOptions } from '@/types/instance'

function Header() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  const handleCreateInstance = () => {
    setIsModalOpen(true)
  }

  const handleSaveInstance = (name: string, options: CreateInstanceOptions) => {
    // TODO: Implement API call to create instance
    console.log('Creating instance:', { name, options })
    // For now, just log the data - you'll implement the API call later
  }

  return (
    <>
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto max-w-4xl px-4 py-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-bold text-gray-900">
              LlamaCtl Dashboard
            </h1>
            
            <Button onClick={handleCreateInstance}>
              Create Instance
            </Button>
          </div>
        </div>
      </header>

      <InstanceModal
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        onSave={handleSaveInstance}
      />
    </>
  )
}

export default Header