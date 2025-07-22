import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Instance } from '@/types/instance'
import { Edit, FileText, Play, Square, Trash2 } from 'lucide-react'

interface InstanceCardProps {
  instance: Instance
  startInstance: (name: string) => void
  stopInstance: (name: string) => void
  deleteInstance: (name: string) => void
  editInstance: (instance: Instance) => void
}

function InstanceCard({ instance, startInstance, stopInstance, deleteInstance, editInstance }: InstanceCardProps) {

  const handleStart = () => {
    startInstance(instance.name)
  }

  const handleStop = () => {
    stopInstance(instance.name)
  }

  const handleDelete = () => {
    if (confirm(`Are you sure you want to delete instance "${instance.name}"?`)) {
      deleteInstance(instance.name)
    }
  }

  const handleEdit = () => {
    editInstance(instance)
  }

  const handleLogs = () => {
    // Logic for viewing logs (e.g., open a logs page)
    console.log(`View logs for instance: ${instance.name}`)
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{instance.name}</CardTitle>
          <Badge variant={instance.running ? "default" : "secondary"}>
            {instance.running ? "Running" : "Stopped"}
          </Badge>
        </div>
      </CardHeader>
      
      <CardContent>
        <div className="flex gap-1">
          <Button 
            size="sm" 
            variant="outline"
            onClick={handleStart}
            disabled={instance.running}
            title="Start instance"
          >
            <Play className="h-4 w-4" />
          </Button>
          
          <Button 
            size="sm" 
            variant="outline" 
            onClick={handleStop}
            disabled={!instance.running}
            title="Stop instance"
          >
            <Square className="h-4 w-4" />
          </Button>
          
          <Button 
            size="sm" 
            variant="outline" 
            onClick={handleEdit}
            title="Edit instance"
          >
            <Edit className="h-4 w-4" />
          </Button>
          
          <Button 
            size="sm" 
            variant="outline" 
            onClick={handleLogs}
            title="View logs"
          >
            <FileText className="h-4 w-4" />
          </Button>
          
          <Button 
            size="sm" 
            variant="destructive" 
            onClick={handleDelete}
            disabled={instance.running}
            title="Delete instance"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default InstanceCard