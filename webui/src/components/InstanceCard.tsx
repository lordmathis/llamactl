// ui/src/components/InstanceCard.tsx
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Instance } from "@/types/instance";
import { Edit, FileText, Play, Square, Trash2 } from "lucide-react";
import LogsModal from "@/components/LogModal";
import HealthBadge from "@/components/HealthBadge";
import { useState } from "react";
import { useInstanceHealth } from "@/hooks/useInstanceHealth";

interface InstanceCardProps {
  instance: Instance;
  startInstance: (name: string) => void;
  stopInstance: (name: string) => void;
  deleteInstance: (name: string) => void;
  editInstance: (instance: Instance) => void;
}

function InstanceCard({
  instance,
  startInstance,
  stopInstance,
  deleteInstance,
  editInstance,
}: InstanceCardProps) {
  const [isLogsOpen, setIsLogsOpen] = useState(false);
  const health = useInstanceHealth(instance.name, instance.running);

  const handleStart = () => {
    startInstance(instance.name);
  };

  const handleStop = () => {
    stopInstance(instance.name);
  };

  const handleDelete = () => {
    if (
      confirm(`Are you sure you want to delete instance "${instance.name}"?`)
    ) {
      deleteInstance(instance.name);
    }
  };

  const handleEdit = () => {
    editInstance(instance);
  };

  const handleLogs = () => {
    setIsLogsOpen(true);
  };

  return (
    <>
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">{instance.name}</CardTitle>
            {instance.running && <HealthBadge health={health} />}
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
              data-testid="start-instance-button"
            >
              <Play className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              variant="outline"
              onClick={handleStop}
              disabled={!instance.running}
              title="Stop instance"
              data-testid="stop-instance-button"
            >
              <Square className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              variant="outline"
              onClick={handleEdit}
              title="Edit instance"
              data-testid="edit-instance-button"
            >
              <Edit className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              variant="outline"
              onClick={handleLogs}
              title="View logs"
              data-testid="view-logs-button"
            >
              <FileText className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              variant="destructive"
              onClick={handleDelete}
              disabled={instance.running}
              title="Delete instance"
              data-testid="delete-instance-button"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      <LogsModal
        open={isLogsOpen}
        onOpenChange={setIsLogsOpen}
        instanceName={instance.name}
        isRunning={instance.running}
      />
    </>
  );
}

export default InstanceCard;