import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { BackendType, type BackendTypeValue, type CreateInstanceOptions } from "@/types/instance";
import { backendsApi } from "@/lib/api";
import { toast } from "sonner";

interface ParseCommandDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onParsed: (options: CreateInstanceOptions) => void;
}

const ParseCommandDialog: React.FC<ParseCommandDialogProps> = ({
  open,
  onOpenChange,
  onParsed,
}) => {
  const [command, setCommand] = useState('');
  const [backendType, setBackendType] = useState<BackendTypeValue>(BackendType.LLAMA_CPP);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleParse = async () => {
    if (!command.trim()) {
      setError("Command cannot be empty");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      let options: CreateInstanceOptions;

      // Parse based on selected backend type
      switch (backendType) {
        case BackendType.LLAMA_CPP:
          options = await backendsApi.llamaCpp.parseCommand(command);
          break;
        case BackendType.MLX_LM:
          options = await backendsApi.mlx.parseCommand(command);
          break;
        case BackendType.VLLM:
          options = await backendsApi.vllm.parseCommand(command);
          break;
        default:
          throw new Error(`Unsupported backend type: ${backendType}`);
      }

      onParsed(options);
      onOpenChange(false);
      setCommand('');
      setError(null);
      toast.success('Command parsed successfully');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to parse command';
      setError(errorMessage);
      toast.error('Failed to parse command', {
        description: errorMessage
      });
    } finally {
      setLoading(false);
    }
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setCommand('');
      setBackendType(BackendType.LLAMA_CPP);
      setError(null);
    }
    onOpenChange(open);
  };

  const backendPlaceholders: Record<BackendTypeValue, string> = {
    [BackendType.LLAMA_CPP]: "llama-server --model /path/to/model.gguf --gpu-layers 32 --ctx-size 4096",
    [BackendType.MLX_LM]: "mlx_lm.server --model mlx-community/Mistral-7B-Instruct-v0.3-4bit --host 0.0.0.0 --port 8080",
    [BackendType.VLLM]: "vllm serve microsoft/DialoGPT-medium --tensor-parallel-size 2 --gpu-memory-utilization 0.9",
  };

  const getPlaceholderForBackend = (backendType: BackendTypeValue): string => {
    return backendPlaceholders[backendType] || "Enter your command here...";
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Parse Backend Command</DialogTitle>
          <DialogDescription>
            Select your backend type and paste the command to automatically populate the form fields
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="backend-type">Backend Type</Label>
            <select
              id="backend-type"
              value={backendType}
              onChange={(e) => setBackendType(e.target.value as BackendTypeValue)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value={BackendType.LLAMA_CPP}>Llama Server</option>
              <option value={BackendType.MLX_LM}>MLX LM</option>
              <option value={BackendType.VLLM}>vLLM</option>
            </select>
          </div>

          <div>
            <Label htmlFor="command">Command</Label>
            <textarea
              id="command"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder={getPlaceholderForBackend(backendType)}
              className="w-full h-32 p-3 mt-2 border border-input rounded-md font-mono text-sm resize-vertical focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
            />
          </div>

          {error && (
            <div className="text-destructive text-sm bg-destructive/10 p-3 rounded-md">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button 
            onClick={() => {
              handleParse().catch(console.error);
            }}
            disabled={!command.trim() || loading}
          >
            {loading ? 'Parsing...' : 'Parse Command'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ParseCommandDialog;