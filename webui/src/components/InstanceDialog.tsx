import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { BackendType, type CreateInstanceOptions, type Instance } from "@/types/instance";
import type { BackendOptions } from "@/schemas/instanceOptions";
import ParseCommandDialog from "@/components/ParseCommandDialog";
import InstanceSettingsCard from "@/components/instance/InstanceSettingsCard";
import BackendConfigurationCard from "@/components/instance/BackendConfigurationCard";
import { Upload } from "lucide-react";
import { useInstanceDefaults } from "@/contexts/ConfigContext";

interface InstanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (name: string, options: CreateInstanceOptions) => void;
  instance?: Instance; // For editing existing instance
}

const InstanceDialog: React.FC<InstanceDialogProps> = ({
  open,
  onOpenChange,
  onSave,
  instance,
}) => {
  const isEditing = !!instance;
  const instanceDefaults = useInstanceDefaults();

  const [instanceName, setInstanceName] = useState("");
  const [formData, setFormData] = useState<CreateInstanceOptions>({});
  const [nameError, setNameError] = useState("");
  const [showParseDialog, setShowParseDialog] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);


  // Reset form when dialog opens/closes or when instance changes
  useEffect(() => {
    if (open) {
      if (instance) {
        // Populate form with existing instance data
        setInstanceName(instance.name);
        setFormData(instance.options || {});
      } else {
        // Reset form for new instance with defaults from config
        setInstanceName("");
        setFormData({
          auto_restart: instanceDefaults?.autoRestart ?? true,
          max_restarts: instanceDefaults?.maxRestarts,
          restart_delay: instanceDefaults?.restartDelay,
          on_demand_start: instanceDefaults?.onDemandStart,
          backend_type: BackendType.LLAMA_CPP, // Default backend type
          backend_options: {},
        });
      }
      setNameError(""); // Reset any name errors
    }
  }, [open, instance, instanceDefaults]);

  const handleFieldChange = (key: keyof CreateInstanceOptions, value: unknown) => {
    setFormData((prev) => {
      // If backend_type is changing, clear backend_options
      if (key === 'backend_type' && prev.backend_type !== value) {
        return {
          ...prev,
          backend_type: value as CreateInstanceOptions['backend_type'],
          backend_options: {}, // Clear backend options when backend type changes
        };
      }

      return {
        ...prev,
        [key]: value,
      } as CreateInstanceOptions;
    });
  };

  const handleBackendFieldChange = (key: string, value: unknown) => {
    setFormData((prev) => ({
      ...prev,
      backend_options: {
        ...prev.backend_options,
        [key]: value,
      } as BackendOptions,
    }));
  };

  const handleNameChange = (name: string) => {
    setInstanceName(name);
    // Validate instance name
    if (!name.trim()) {
      setNameError("Instance name is required");
    } else if (!/^[a-zA-Z0-9-_]+$/.test(name)) {
      setNameError(
        "Instance name can only contain letters, numbers, hyphens, and underscores"
      );
    } else {
      setNameError("");
    }
  };

  const handleSave = () => {
    // Validate instance name before saving
    if (!instanceName.trim()) {
      setNameError("Instance name is required");
      return;
    }

    // Validate docker_enabled and command_override relationship
    if (formData.backend_type !== BackendType.MLX_LM) {
      if (formData.docker_enabled === true && formData.command_override) {
        setNameError("Command override cannot be set when Docker is enabled");
        return;
      }
    }

    // Clean up undefined values to avoid sending empty fields
    const cleanOptions: CreateInstanceOptions = {} as CreateInstanceOptions;
    Object.entries(formData).forEach(([key, value]) => {
      const typedKey = key as keyof CreateInstanceOptions;

      if (key === 'backend_options' && value && typeof value === 'object' && !Array.isArray(value)) {
        // Handle backend_options specially - clean nested object
        const cleanBackendOptions: Record<string, unknown> = {};
        Object.entries(value).forEach(([backendKey, backendValue]) => {
          if (backendValue !== undefined && backendValue !== null && (typeof backendValue !== 'string' || backendValue.trim() !== "")) {
            // Handle arrays - don't include empty arrays
            if (Array.isArray(backendValue) && backendValue.length === 0) {
              return;
            }
            cleanBackendOptions[backendKey] = backendValue;
          }
        });

        // Only include backend_options if it has content
        if (Object.keys(cleanBackendOptions).length > 0) {
          (cleanOptions as Record<string, unknown>)[typedKey] = cleanBackendOptions as BackendOptions;
        }
      } else if (value !== undefined && value !== null) {
        // Skip empty strings
        if (typeof value === 'string' && value.trim() === "") {
          return;
        }
        // Skip empty arrays
        if (Array.isArray(value) && value.length === 0) {
          return;
        }
        (cleanOptions as Record<string, unknown>)[typedKey] = value;
      }
    });

    onSave(instanceName, cleanOptions);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };


  const handleCommandParsed = (parsedOptions: CreateInstanceOptions) => {
    setFormData(prev => ({
      ...prev,
      ...parsedOptions,
    }));
    setShowParseDialog(false);
  };

  const handleImportFile = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string;
        const importedData = JSON.parse(content) as { name?: string; options?: CreateInstanceOptions };

        // Validate that it's an instance export
        if (!importedData.name || !importedData.options) {
          alert('Invalid instance file: Missing required fields (name, options)');
          return;
        }

        // Set the instance name (only for new instances, not editing)
        if (!isEditing && typeof importedData.name === 'string') {
          handleNameChange(importedData.name);
        }

        // Populate all the options from the imported file
        if (importedData.options) {
          setFormData(prev => ({
            ...prev,
            ...importedData.options,
          }));
        }

        // Reset the file input
        event.target.value = '';
      } catch (error) {
        console.error('Failed to parse instance file:', error);
        alert(`Failed to parse instance file: ${error instanceof Error ? error.message : 'Invalid JSON'}`);
      }
    };

    reader.readAsText(file);
  };

  // Save button label logic
  let saveButtonLabel = "Create Instance";
  if (isEditing) {
    if (instance?.status === "running") {
      saveButtonLabel = "Update & Restart Instance";
    } else {
      saveButtonLabel = "Update Instance";
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <DialogTitle>
                {isEditing ? "Edit Instance" : "Create New Instance"}
              </DialogTitle>
              <DialogDescription>
                {isEditing
                  ? "Modify the instance configuration below."
                  : "Configure your new llama-server instance below."}
              </DialogDescription>
            </div>
            {!isEditing && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleImportFile}
                title="Import instance configuration from JSON file"
                className="ml-2"
              >
                <Upload className="h-4 w-4 mr-2" />
                Import
              </Button>
            )}
          </div>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            onChange={handleFileChange}
            className="hidden"
          />
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          <div className="space-y-6 py-4">
            {/* Instance Settings Card */}
            <InstanceSettingsCard
              instanceName={instanceName}
              nameError={nameError}
              isEditing={isEditing}
              formData={formData}
              onNameChange={handleNameChange}
              onChange={handleFieldChange}
            />

            {/* Backend Configuration Card */}
            <BackendConfigurationCard
              formData={formData}
              onBackendFieldChange={handleBackendFieldChange}
              onChange={handleFieldChange}
              onParseCommand={() => setShowParseDialog(true)}
            />

          </div>
        </div>

        <DialogFooter className="pt-4 border-t">
          <Button
            variant="outline"
            onClick={handleCancel}
            data-testid="dialog-cancel-button"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={!instanceName.trim() || !!nameError}
            data-testid="dialog-save-button"
          >
            {saveButtonLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
      
      <ParseCommandDialog
        open={showParseDialog}
        onOpenChange={setShowParseDialog}
        onParsed={handleCommandParsed}
        backendType={formData.backend_type || BackendType.LLAMA_CPP}
      />
    </Dialog>
  );
};

export default InstanceDialog;
