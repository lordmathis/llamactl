// ui/src/components/InstanceCard.tsx
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { Instance } from "@/types/instance";
import { Edit, FileText, Play, Square, Trash2, MoreHorizontal, Download, Boxes } from "lucide-react";
import LogsDialog from "@/components/LogDialog";
import ModelsDialog from "@/components/ModelsDialog";
import HealthBadge from "@/components/HealthBadge";
import BackendBadge from "@/components/BackendBadge";
import { useState, useEffect } from "react";
import { useInstanceHealth } from "@/hooks/useInstanceHealth";
import { instancesApi, llamaCppApi, type Model } from "@/lib/api";

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
  const [isModelsOpen, setIsModelsOpen] = useState(false);
  const [showAllActions, setShowAllActions] = useState(false);
  const [models, setModels] = useState<Model[]>([]);
  const health = useInstanceHealth(instance.name, instance.status);

  const running = instance.status === "running";
  const isLlamaCpp = instance.options?.backend_type === "llama_cpp";

  // Fetch models for llama.cpp instances
  useEffect(() => {
    if (!isLlamaCpp || !running) {
      setModels([]);
      return;
    }

    void (async () => {
      try {
        const fetchedModels = await llamaCppApi.getModels(instance.name);
        setModels(fetchedModels);
      } catch {
        setModels([]);
      }
    })();
  }, [instance.name, isLlamaCpp, running]);

  // Calculate model counts
  const totalModels = models.length;
  const loadedModels = models.filter(m => m.status?.value === "loaded").length;

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

  const handleModels = () => {
    setIsModelsOpen(true);
  };

  const handleExport = () => {
    void (async () => {
      try {
        // Fetch the most up-to-date instance data from the backend
        const instanceData = await instancesApi.get(instance.name);

        // Convert to JSON string with pretty formatting (matching backend format)
        const jsonString = JSON.stringify(instanceData, null, 2);

        // Create a blob and download link
        const blob = new Blob([jsonString], { type: "application/json" });
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `${instance.name}.json`;

        // Trigger download
        document.body.appendChild(link);
        link.click();

        // Cleanup
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
      } catch (error) {
        console.error("Failed to export instance:", error);
        alert(`Failed to export instance: ${error instanceof Error ? error.message : "Unknown error"}`);
      }
    })();
  };

  return (
    <>
      <Card className="hover:shadow-md transition-shadow">
        <CardHeader className="pb-4">
          {/* Header with instance name and status badges */}
          <div className="space-y-3">
            <CardTitle className="text-lg font-semibold leading-tight break-words">
              {instance.name}
            </CardTitle>
            
            {/* Badges row */}
            <div className="flex items-center gap-2 flex-wrap">
              <BackendBadge backend={instance.options?.backend_type} docker={instance.options?.docker_enabled} />
              {running && <HealthBadge health={health} />}
              {isLlamaCpp && running && totalModels > 1 && (
                <Badge variant="secondary" className="text-xs">
                  <Boxes className="h-3 w-3 mr-1" />
                  {loadedModels}/{totalModels} models
                </Badge>
              )}
            </div>
          </div>
        </CardHeader>

        <CardContent className="pt-0">
          {/* Primary actions - always visible */}
          <div className="flex items-center gap-2 mb-3">
            <Button
              size="sm"
              variant={running ? "outline" : "default"}
              onClick={running ? handleStop : handleStart}
              className="flex-1"
              title={running ? "Stop instance" : "Start instance"}
              data-testid={running ? "stop-instance-button" : "start-instance-button"}
            >
              {running ? (
                <>
                  <Square className="h-4 w-4 mr-1" />
                  Stop
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-1" />
                  Start
                </>
              )}
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
              onClick={() => setShowAllActions(!showAllActions)}
              title="More actions"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </div>

          {/* Secondary actions - collapsible */}
          {showAllActions && (
            <div className="flex items-center gap-2 pt-2 border-t border-border flex-wrap">
              <Button
                size="sm"
                variant="outline"
                onClick={handleLogs}
                title="View logs"
                data-testid="view-logs-button"
              >
                <FileText className="h-4 w-4 mr-1" />
                Logs
              </Button>

              {isLlamaCpp && totalModels > 1 && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleModels}
                  title="Manage models"
                  data-testid="manage-models-button"
                >
                  <Boxes className="h-4 w-4 mr-1" />
                  Models
                </Button>
              )}

              <Button
                size="sm"
                variant="outline"
                onClick={handleExport}
                title="Export instance"
                data-testid="export-instance-button"
              >
                <Download className="h-4 w-4 mr-1" />
                Export
              </Button>

              <Button
                size="sm"
                variant="destructive"
                onClick={handleDelete}
                disabled={running}
                title="Delete instance"
                data-testid="delete-instance-button"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <LogsDialog
        open={isLogsOpen}
        onOpenChange={setIsLogsOpen}
        instanceName={instance.name}
        isRunning={running}
      />

      <ModelsDialog
        open={isModelsOpen}
        onOpenChange={setIsModelsOpen}
        instanceName={instance.name}
        isRunning={running}
      />
    </>
  );
}

export default InstanceCard;