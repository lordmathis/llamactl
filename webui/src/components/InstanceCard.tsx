// ui/src/components/InstanceCard.tsx
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { Instance } from "@/types/instance";
import { Edit, ExternalLink, FileText, Play, Square, Trash2, MoreHorizontal, Download, Boxes, Layers, MessageSquare, Cpu } from "lucide-react";
import LogsDialog from "@/components/LogDialog";
import ModelsDialog from "@/components/ModelsDialog";
import ChatDialog from "@/components/ChatDialog";
import HealthBadge from "@/components/HealthBadge";
import BackendBadge from "@/components/BackendBadge";
import { useState, useEffect, useCallback, useRef } from "react";
import { useInstanceHealth } from "@/hooks/useInstanceHealth";
import { instancesApi, llamaCppApi, type Model } from "@/lib/api";

const MODELS_POLL_INTERVAL = 10_000; // 10 seconds

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
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [showAllActions, setShowAllActions] = useState(false);
  const [models, setModels] = useState<Model[]>([]);
  const health = useInstanceHealth(instance.name, instance.status);
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const running = instance.status === "running";
  const isLlamaCpp = instance.options?.backend_type === "llama_cpp";

  // Fetch models for llama.cpp instances
  const fetchModels = useCallback(async () => {
    if (!isLlamaCpp || !running) {
      setModels([]);
      return;
    }
    try {
      const fetchedModels = await llamaCppApi.getModels(instance.name);
      setModels(fetchedModels);
    } catch {
      setModels([]);
    }
  }, [instance.name, isLlamaCpp, running]);

  // Poll model state while running so the badge stays current
  useEffect(() => {
    void fetchModels();

    if (isLlamaCpp && running) {
      pollIntervalRef.current = setInterval(() => {
        void fetchModels();
      }, MODELS_POLL_INTERVAL);
    }

    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
        pollIntervalRef.current = null;
      }
    };
  }, [fetchModels, isLlamaCpp, running]);

  // Callback passed to ModelsDialog so a load/unload immediately refreshes the card
  const handleModelsChange = useCallback(() => {
    void fetchModels();
  }, [fetchModels]);

  // Calculate model counts
  const totalModels = models.length;
  const loadedModels = models.filter(m => m.status?.value === "loaded").length;

  // For single-model instances show the model id/alias directly
  const singleLoadedModel =
    isLlamaCpp && running && totalModels === 1 && loadedModels === 1
      ? models[0].id
      : null;

  // Models available for chat (loaded ones only)
  const chatModels = models.filter(m => m.status?.value === 'loaded');
  const canChat = running && isLlamaCpp && chatModels.length > 0;

  // Configured model from instance options (shown when stopped)
  const configuredModel = (() => {
    const opts = instance.options?.backend_options as Record<string, unknown> | undefined;
    return (opts?.hf_repo as string) || (opts?.model as string) || null;
  })();

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
              {instance.options?.group && (
                <Badge variant="outline" className="text-xs">
                  <Layers className="h-3 w-3 mr-1" />
                  {instance.options.group}
                </Badge>
              )}
              {isLlamaCpp && running && singleLoadedModel && (
                <Badge variant="secondary" className="text-xs max-w-[14rem] truncate" title={singleLoadedModel}>
                  {singleLoadedModel}
                </Badge>
              )}
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
          {/* Model info line */}
          {isLlamaCpp && (
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-3 min-w-0">
              <Cpu className="h-3 w-3 shrink-0" />
              {running && singleLoadedModel && (
                <span className="truncate" title={singleLoadedModel}>{singleLoadedModel}</span>
              )}
              {running && totalModels > 1 && (
                <span>{loadedModels}/{totalModels} models loaded</span>
              )}
              {running && totalModels === 0 && (
                <span className="italic">No models loaded</span>
              )}
              {!running && configuredModel && (
                <span className="truncate" title={configuredModel}>{configuredModel}</span>
              )}
              {!running && !configuredModel && (
                <span className="italic">Router mode</span>
              )}
            </div>
          )}

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

            {canChat && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setIsChatOpen(true)}
                title="Chat with loaded model"
                data-testid="chat-button"
              >
                <MessageSquare className="h-4 w-4" />
              </Button>
            )}

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

              {isLlamaCpp && running && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => window.open(`${document.baseURI}api/v1/instances/${instance.name}/proxy/`, "_blank")}
                  title="Open llama-server UI"
                  data-testid="open-llama-ui-button"
                >
                  <ExternalLink className="h-4 w-4 mr-1" />
                  Server UI
                </Button>
              )}

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
        onOpenChange={(open) => {
          setIsModelsOpen(open);
          // Refresh card badge immediately when dialog closes
          if (!open) handleModelsChange();
        }}
        instanceName={instance.name}
        isRunning={running}
        onModelsChange={handleModelsChange}
      />

      <ChatDialog
        open={isChatOpen}
        onOpenChange={setIsChatOpen}
        instanceName={instance.name}
        loadedModels={chatModels}
      />
    </>
  );
}

export default InstanceCard;