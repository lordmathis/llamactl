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
import type { CreateInstanceOptions, Instance } from "@/types/instance";
import { getBasicFields, getAdvancedFields } from "@/lib/zodFormUtils";
import { ChevronDown, ChevronRight } from "lucide-react";
import ZodFormField from "@/components/ZodFormField";

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

  // Get field lists dynamically from the type
  const basicFields = getBasicFields();
  const advancedFields = getAdvancedFields();

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
        });
      }
      setShowAdvanced(false); // Always start with basic view
      setNameError(""); // Reset any name errors
    }
  }, [open, instance]);

  const handleFieldChange = (key: keyof CreateInstanceOptions, value: any) => {
    setFormData((prev) => ({
      ...prev,
      [key]: value,
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
      if (value !== undefined && value !== "" && value !== null) {
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

  // Check if auto_restart is enabled
  const isAutoRestartEnabled = formData.auto_restart === true;

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
            <div className="space-y-4">
              <h3 className="text-lg font-medium">
                Auto Restart Configuration
              </h3>

              {/* Auto Restart Toggle */}
              <ZodFormField
                fieldKey="auto_restart"
                value={formData.auto_restart}
                onChange={handleFieldChange}
              />

              {/* Show restart options only when auto restart is enabled */}
              {isAutoRestartEnabled && (
                <div className="ml-6 space-y-4 border-l-2 border-muted pl-4">
                  <ZodFormField
                    fieldKey="max_restarts"
                    value={formData.max_restarts}
                    onChange={handleFieldChange}
                  />
                  <ZodFormField
                    fieldKey="restart_delay"
                    value={formData.restart_delay}
                    onChange={handleFieldChange}
                  />
                </div>
              )}
            </div>

            {/* Basic Fields - Automatically generated from type (excluding auto restart options) */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium">Basic Configuration</h3>
              {basicFields
                .filter(
                  (fieldKey) =>
                    fieldKey !== "auto_restart" &&
                    fieldKey !== "max_restarts" &&
                    fieldKey !== "restart_delay"
                ) // Exclude auto_restart, max_restarts, and restart_delay as they're handled above
                .map((fieldKey) => (
                  <ZodFormField
                    key={fieldKey}
                    fieldKey={fieldKey}
                    value={formData[fieldKey]}
                    onChange={handleFieldChange}
                  />
                ))}
            </div>

            {/* Advanced Fields Toggle */}
            <div className="border-t pt-4">
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
                        !["max_restarts", "restart_delay"].includes(f as string)
                    ).length
                  }{" "}
                  options)
                </span>
              </Button>
            </div>

            {/* Advanced Fields - Automatically generated from type (excluding restart options) */}
            {showAdvanced && (
              <div className="space-y-4 pl-6 border-l-2 border-muted">
                <div className="space-y-4">
                  {advancedFields
                    .filter(
                      (fieldKey) =>
                        !["max_restarts", "restart_delay"].includes(
                          fieldKey as string
                        )
                    ) // Exclude restart options as they're handled above
                    .sort()
                    .map((fieldKey) => (
                      <ZodFormField
                        key={fieldKey}
                        fieldKey={fieldKey}
                        value={formData[fieldKey]}
                        onChange={handleFieldChange}
                      />
                    ))}
                </div>
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
    </Dialog>
  );
};

export default InstanceDialog;
