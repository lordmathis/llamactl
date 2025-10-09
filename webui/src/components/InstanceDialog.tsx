import React, { useState, useEffect } from "react";
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
import ParseCommandDialog from "@/components/ParseCommandDialog";
import InstanceSettingsCard from "@/components/instance/InstanceSettingsCard";
import BackendConfigurationCard from "@/components/instance/BackendConfigurationCard";

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

  const [instanceName, setInstanceName] = useState("");
  const [formData, setFormData] = useState<CreateInstanceOptions>({});
  const [nameError, setNameError] = useState("");
  const [showParseDialog, setShowParseDialog] = useState(false);


  // Reset form when dialog opens/closes or when instance changes
  useEffect(() => {
    if (open) {
      if (instance) {
        // Populate form with existing instance data
        setInstanceName(instance.name);
        setFormData(instance.options || {});
      } else {
        // Reset form for new instance
        setInstanceName("");
        setFormData({
          auto_restart: true, // Default value
          backend_type: BackendType.LLAMA_CPP, // Default backend type
          backend_options: {},
        });
      }
      setNameError(""); // Reset any name errors
    }
  }, [open, instance]);

  const handleFieldChange = (key: keyof CreateInstanceOptions, value: any) => {
    setFormData((prev) => {
      // If backend_type is changing, clear backend_options
      if (key === 'backend_type' && prev.backend_type !== value) {
        return {
          ...prev,
          [key]: value,
          backend_options: {}, // Clear backend options when backend type changes
        };
      }
      
      return {
        ...prev,
        [key]: value,
      };
    });
  };

  const handleBackendFieldChange = (key: string, value: any) => {
    setFormData((prev) => ({
      ...prev,
      backend_options: {
        ...prev.backend_options,
        [key]: value,
      } as any,
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

    // Clean up undefined values to avoid sending empty fields
    const cleanOptions: CreateInstanceOptions = {};
    Object.entries(formData).forEach(([key, value]) => {
      if (key === 'backend_options' && value && typeof value === 'object' && !Array.isArray(value)) {
        // Handle backend_options specially - clean nested object
        const cleanBackendOptions: any = {};
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
          (cleanOptions as any)[key] = cleanBackendOptions;
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
        (cleanOptions as any)[key] = value;
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
          <DialogTitle>
            {isEditing ? "Edit Instance" : "Create New Instance"}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Modify the instance configuration below."
              : "Configure your new llama-server instance below."}
          </DialogDescription>
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
