import { Button } from '@/components/ui/button'

function Header() {
  const handleCreateInstance = () => {
    // TODO: Open create instance dialog
    console.log('Create instance clicked')
  }

  return (
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
  )
}

export default Header