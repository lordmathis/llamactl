import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2 } from "lucide-react";
import { apiKeysApi } from "@/lib/api";
import { CreateKeyRequest, PermissionMode, InstancePermission } from "@/types/apiKey";
import { useInstances } from "@/contexts/InstancesContext";
import { format, addDays } from "date-fns";

interface CreateApiKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onKeyCreated: (plainTextKey: string) => void;
}

function CreateApiKeyDialog({ open, onOpenChange, onKeyCreated }: CreateApiKeyDialogProps) {
  const { instances } = useInstances();
  const [name, setName] = useState("");
  const [permissionMode, setPermissionMode] = useState<PermissionMode>(PermissionMode.AllowAll);
  const [expiresAt, setExpiresAt] = useState<string>("");
  const [instancePermissions, setInstancePermissions] = useState<Record<number, boolean>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const formatDisplayDate = (dateString: string) => {
    if (!dateString) return null;
    try {
      const date = new Date(dateString);
      return format(date, "d MMMM yyyy");
    } catch {
      return null;
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!name.trim()) {
      setError("Name is required");
      return;
    }

    if (name.length > 100) {
      setError("Name must be 100 characters or less");
      return;
    }

    if (permissionMode === PermissionMode.PerInstance) {
      const hasAnyPermission = Object.values(instancePermissions).some(v => v);
      if (!hasAnyPermission) {
        setError("At least one instance permission is required for per-instance mode");
        return;
      }
    }

    // Build request
    const permissions: InstancePermission[] = [];
    if (permissionMode === PermissionMode.PerInstance) {
      Object.entries(instancePermissions).forEach(([instanceId, canInfer]) => {
        if (canInfer) {
          permissions.push({
            InstanceID: parseInt(instanceId),
            CanInfer: true,
          });
        }
      });
    }

    const request: CreateKeyRequest = {
      Name: name.trim(),
      PermissionMode: permissionMode,
      InstancePermissions: permissions,
    };

    // Add expiration if provided
    if (expiresAt) {
      const expirationDate = new Date(expiresAt);
      const now = new Date();
      if (expirationDate <= now) {
        setError("Expiration date must be in the future");
        return;
      }
      request.ExpiresAt = Math.floor(expirationDate.getTime() / 1000);
    }

    setLoading(true);
    try {
      const response = await apiKeysApi.create(request);
      onKeyCreated(response.key);
      // Reset form
      setName("");
      setPermissionMode(PermissionMode.AllowAll);
      setExpiresAt("");
      setInstancePermissions({});
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create API key");
    } finally {
      setLoading(false);
    }
  };

  const handleInstancePermissionChange = (instanceId: number, checked: boolean) => {
    setInstancePermissions({
      ...instancePermissions,
      [instanceId]: checked,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Create API Key</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My API Key"
              maxLength={100}
              disabled={loading}
            />
          </div>

          <div className="space-y-3">
            <Label>Permission Mode</Label>
            <RadioGroup
              value={permissionMode}
              onValueChange={(value) => setPermissionMode(value as PermissionMode)}
              disabled={loading}
            >
              <div className="flex items-center space-x-2">
                <RadioGroupItem value={PermissionMode.AllowAll} id="allow-all" />
                <Label htmlFor="allow-all" className="font-normal cursor-pointer">
                  Full Access
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value={PermissionMode.PerInstance} id="per-instance" />
                <Label htmlFor="per-instance" className="font-normal cursor-pointer">
                  Per-Instance Access
                </Label>
              </div>
            </RadioGroup>

            {permissionMode === PermissionMode.AllowAll && (
              <p className="text-sm text-muted-foreground">
                This key will have access to all instances
              </p>
            )}

            {permissionMode === PermissionMode.PerInstance && (
              <div className="space-y-2 border rounded-lg p-4">
                <Label className="text-sm font-semibold">Instance Permissions</Label>
                {instances.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No instances available</p>
                ) : (
                  <div className="space-y-2">
                    {instances.map((instance) => (
                      <div key={instance.id} className="flex items-center space-x-2">
                        <Checkbox
                          id={`instance-${instance.id}`}
                          checked={instancePermissions[instance.id] || false}
                          onCheckedChange={(checked) =>
                            handleInstancePermissionChange(instance.id, checked as boolean)
                          }
                          disabled={loading}
                        />
                        <Label
                          htmlFor={`instance-${instance.id}`}
                          className="font-normal cursor-pointer flex-1"
                        >
                          {instance.name}
                        </Label>
                        <span className="text-sm text-muted-foreground">Can Infer</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="expires-at">Expiration Date (Optional)</Label>
            <Input
              id="expires-at"
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              disabled={loading}
            />
            {expiresAt && formatDisplayDate(expiresAt) && (
              <p className="text-sm text-muted-foreground">
                Expires on {formatDisplayDate(expiresAt)}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default CreateApiKeyDialog;
