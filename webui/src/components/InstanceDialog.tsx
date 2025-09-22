import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { BackendType, type CreateInstanceOptions, type Instance } from "@/types/instance";
import { getAdvancedFields, getAdvancedBackendFields } from "@/lib/zodFormUtils";
import { ChevronDown, ChevronRight, Terminal } from "lucide-react";
import ParseCommandDialog from "@/components/ParseCommandDialog";
import AutoRestartConfiguration from "@/components/instance/AutoRestartConfiguration";
import BasicInstanceFields from "@/components/instance/BasicInstanceFields";
import BackendConfiguration from "@/components/instance/BackendConfiguration";
import AdvancedInstanceFields from "@/components/instance/AdvancedInstanceFields";

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
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [nameError, setNameError] = useState("");
  const [showParseDialog, setShowParseDialog] = useState(false);

  // Get field lists dynamically from the type
  const advancedFields = getAdvancedFields();
  const advancedBackendFields = getAdvancedBackendFields(formData.backend_type);

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
      setShowAdvanced(false); // Always start with basic view
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
      if (key === 'backend_options' && value && typeof value === 'object') {
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
      } else if (value !== undefined && value !== null && (typeof value !== 'string' || value.trim() !== "")) {
        // Handle arrays - don't include empty arrays
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

  const toggleAdvanced = () => {
    setShowAdvanced(!showAdvanced);
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
          <div className="grid gap-6 py-4">
            {/* Instance Name - Special handling since it's not in CreateInstanceOptions */}
            <div className="grid gap-2">
              <Label htmlFor="name">
                Instance Name <span className="text-red-500">*</span>
              </Label>
              <Input
                id="name"
                value={instanceName}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="my-instance"
                disabled={isEditing} // Don't allow name changes when editing
                className={nameError ? "border-red-500" : ""}
              />
              {nameError && <p className="text-sm text-red-500">{nameError}</p>}
              <p className="text-sm text-muted-foreground">
                Unique identifier for the instance
              </p>
            </div>

            {/* Auto Restart Configuration Section */}
            <AutoRestartConfiguration
              formData={formData}
              onChange={handleFieldChange}
            />

            {/* Basic Fields */}
            <BasicInstanceFields
              formData={formData}
              onChange={handleFieldChange}
            />

            {/* Backend Configuration Section */}
            <BackendConfiguration
              formData={formData}
              onBackendFieldChange={handleBackendFieldChange}
              showAdvanced={showAdvanced}
            />

            {/* Advanced Fields Toggle */}
            <div className="border-t pt-4">
              <div className="flex items-center justify-between">
                <Button
                  variant="outline"
                  onClick={() => setShowParseDialog(true)}
                  className="flex items-center gap-2"
                >
                  <Terminal className="h-4 w-4" />
                  Parse Command
                </Button>
                
                <Button
                  variant="ghost"
                  onClick={toggleAdvanced}
                  className="flex items-center gap-2 p-0 h-auto font-medium"
                >
                  {showAdvanced ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronRight className="h-4 w-4" />
                  )}
                  Advanced Configuration
                  <span className="text-muted-foreground text-sm font-normal">
                    (
                    {
                      advancedFields.filter(
                        (f) =>
                          !["max_restarts", "restart_delay", "backend_options"].includes(f as string)
                      ).length + advancedBackendFields.length
                    }{" "}
                    options)
                  </span>
                </Button>
              </div>
            </div>

            {/* Advanced Fields */}
            {showAdvanced && (
              <div className="space-y-4 pl-6 border-l-2 border-muted">
                <AdvancedInstanceFields
                  formData={formData}
                  onChange={handleFieldChange}
                />
              </div>
            )}
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
      />
    </Dialog>
  );
};

export default InstanceDialog;
