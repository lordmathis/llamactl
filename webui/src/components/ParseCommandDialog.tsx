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
import { type CreateInstanceOptions } from "@/types/instance";
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
      const options = await backendsApi.llamaCpp.parseCommand(command);
      onParsed(options);
      onOpenChange(false);
      // Reset form
      setCommand('');
      setError(null);
      // Show success toast
      toast.success('Command parsed successfully');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to parse command';
      setError(errorMessage);
      // Show error toast
      toast.error('Failed to parse command', {
        description: errorMessage
      });
    } finally {
      setLoading(false);
    }
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      // Reset form when closing
      setCommand('');
      setError(null);
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Parse Llama Server Command</DialogTitle>
          <DialogDescription>
            Paste your llama-server command to automatically populate the form fields
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div>
            <Label htmlFor="command">Command</Label>
            <textarea
              id="command"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="llama-server --model /path/to/model.gguf --gpu-layers 32 --ctx-size 4096"
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