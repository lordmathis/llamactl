import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Instance } from '@/types/instance'
import { useInstances } from '@/hooks/useInstances'

interface InstanceCardProps {
  instance: Instance
}

function InstanceCard({ instance }: InstanceCardProps) {
  const { startInstance, stopInstance, deleteInstance } = useInstances()

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
        <div className="flex gap-2">
          {!instance.running ? (
            <Button 
              size="sm" 
              onClick={handleStart}
              className="flex-1"
            >
              Start
            </Button>
          ) : (
            <Button 
              size="sm" 
              variant="outline" 
              onClick={handleStop}
              className="flex-1"
            >
              Stop
            </Button>
          )}
          
          <Button 
            size="sm" 
            variant="destructive" 
            onClick={handleDelete}
            disabled={instance.running}
          >
            Delete
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default InstanceCard