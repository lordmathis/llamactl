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
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { BackendType, type CreateInstanceOptions, type Instance } from "@/types/instance";
import type { BackendOptions } from "@/schemas/instanceOptions";
import ParseCommandDialog from "@/components/ParseCommandDialog";
import GeneralTab from "@/components/instance/GeneralTab";
import BackendTab from "@/components/instance/BackendTab";
import AdvancedTab from "@/components/instance/AdvancedTab";
import { Upload } from "lucide-react";
import { useInstanceDefaults, useBackendSettings } from "@/hooks/useConfig";

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
  const [activeTab, setActiveTab] = useState("general");
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Get backend settings for all backends (we'll use this to update docker_enabled on backend type change)
  const llamaCppSettings = useBackendSettings(BackendType.LLAMA_CPP);
  const vllmSettings = useBackendSettings(BackendType.VLLM);
  const mlxSettings = useBackendSettings(BackendType.MLX_LM);

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
          docker_enabled: llamaCppSettings?.dockerEnabled ?? false,
          backend_options: {},
        });
      }
      setNameError(""); // Reset any name errors
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, instance]);

  const handleFieldChange = (key: keyof CreateInstanceOptions, value: unknown) => {
    setFormData((prev) => {
      // If backend_type is changing, update docker_enabled default and clear backend_options
      if (key === 'backend_type' && prev.backend_type !== value) {
        let dockerEnabled = false;
        if (value === BackendType.LLAMA_CPP) {
          dockerEnabled = llamaCppSettings?.dockerEnabled ?? false;
        } else if (value === BackendType.VLLM) {
          dockerEnabled = vllmSettings?.dockerEnabled ?? false;
        } else if (value === BackendType.MLX_LM) {
          dockerEnabled = mlxSettings?.dockerEnabled ?? false;
        }

        return {
          ...prev,
          backend_type: value as CreateInstanceOptions['backend_type'],
          docker_enabled: dockerEnabled,
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

        if (!importedData.name || !importedData.options) {
          alert('Invalid instance file: Missing required fields (name, options)');
          return;
        }

        if (!isEditing && typeof importedData.name === 'string') {
          handleNameChange(importedData.name);
        }

        if (importedData.options) {
          setFormData(prev => ({
            ...prev,
            ...importedData.options,
          }));
        }

        event.target.value = '';
      } catch (error) {
        console.error('Failed to parse instance file:', error);
        alert(`Failed to parse instance file: ${error instanceof Error ? error.message : 'Invalid JSON'}`);
      }
    };

    reader.readAsText(file);
  };

  const tabs = ["general", "backend", "advanced"];

  const handleNext = () => {
    const currentIndex = tabs.indexOf(activeTab);
    if (currentIndex < tabs.length - 1) {
      setActiveTab(tabs[currentIndex + 1]);
    }
  };

  const handleBack = () => {
    const currentIndex = tabs.indexOf(activeTab);
    if (currentIndex > 0) {
      setActiveTab(tabs[currentIndex - 1]);
    }
  };

  const isLastTab = activeTab === tabs[tabs.length - 1];
  const isFirstTab = activeTab === tabs[0];

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
      <DialogContent className="sm:max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
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

        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col min-h-0">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="backend">Backend</TabsTrigger>
            <TabsTrigger value="advanced">Advanced</TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto">
            <TabsContent value="general" className="h-full">
              <GeneralTab
                instanceName={instanceName}
                nameError={nameError}
                isEditing={isEditing}
                formData={formData}
                onNameChange={handleNameChange}
                onChange={handleFieldChange}
              />
            </TabsContent>

            <TabsContent value="backend" className="h-full">
              <BackendTab
                formData={formData}
                onBackendFieldChange={handleBackendFieldChange}
                onChange={handleFieldChange}
                onParseCommand={() => setShowParseDialog(true)}
              />
            </TabsContent>

            <TabsContent value="advanced" className="h-full">
              <AdvancedTab
                formData={formData}
                onBackendFieldChange={handleBackendFieldChange}
              />
            </TabsContent>
          </div>
        </Tabs>

        <DialogFooter className="pt-4 border-t">
          <Button
            variant="outline"
            onClick={handleCancel}
            data-testid="dialog-cancel-button"
          >
            Cancel
          </Button>
          {!isFirstTab && (
            <Button
              variant="outline"
              onClick={handleBack}
            >
              Back
            </Button>
          )}
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex">
                  <Button
                    onClick={isLastTab ? handleSave : handleNext}
                    disabled={!instanceName.trim() || !!nameError}
                    data-testid={isLastTab ? "dialog-save-button" : "dialog-next-button"}
                  >
                    {isLastTab ? saveButtonLabel : "Next"}
                  </Button>
                </span>
              </TooltipTrigger>
              {(!instanceName.trim() || !!nameError) && (
                <TooltipContent>
                  <p>{nameError || "Instance name is required"}</p>
                </TooltipContent>
              )}
            </Tooltip>
          </TooltipProvider>
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
