import { useEffect, useState, Fragment } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Trash2, Copy, Check, X, ChevronDown, ChevronRight } from "lucide-react";
import { apiKeysApi } from "@/lib/api";
import { ApiKey, KeyPermissionResponse, PermissionMode } from "@/types/apiKey";
import CreateApiKeyDialog from "@/components/apikeys/CreateApiKeyDialog";
import { format, formatDistanceToNow } from "date-fns";

function ApiKeysSection() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedRowId, setExpandedRowId] = useState<number | null>(null);
  const [newKeyPlainText, setNewKeyPlainText] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [copiedKey, setCopiedKey] = useState(false);
  const [permissions, setPermissions] = useState<Record<number, KeyPermissionResponse[]>>({});
  const [loadingPermissions, setLoadingPermissions] = useState<Record<number, boolean>>({});

  useEffect(() => {
    fetchKeys();
  }, []);

  const fetchKeys = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiKeysApi.list();
      setKeys(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load API keys");
    } finally {
      setLoading(false);
    }
  };

  const fetchPermissions = async (keyId: number) => {
    if (permissions[keyId]) return;

    setLoadingPermissions({ ...loadingPermissions, [keyId]: true });
    try {
      const data = await apiKeysApi.getPermissions(keyId);
      setPermissions({ ...permissions, [keyId]: data });
    } catch (err) {
      console.error("Failed to load permissions:", err);
    } finally {
      setLoadingPermissions({ ...loadingPermissions, [keyId]: false });
    }
  };

  const handleKeyCreated = (plainTextKey: string) => {
    setNewKeyPlainText(plainTextKey);
    fetchKeys();
    setCreateDialogOpen(false);
  };

  const dismissSuccessBanner = () => {
    setNewKeyPlainText(null);
  };

  const handleCopyKey = async () => {
    if (newKeyPlainText) {
      await navigator.clipboard.writeText(newKeyPlainText);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    }
  };

  const handleDeleteKey = async (id: number, name: string) => {
    if (!confirm(`Are you sure you want to delete the key '${name}'?\n\nThis action cannot be undone.`)) {
      return;
    }

    try {
      await apiKeysApi.delete(id);
      fetchKeys();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete API key");
    }
  };

  const handleRowClick = (key: ApiKey) => {
    if (expandedRowId === key.id) {
      setExpandedRowId(null);
    } else {
      setExpandedRowId(key.id);
      if (key.permission_mode === PermissionMode.PerInstance) {
        fetchPermissions(key.id);
      }
    }
  };

  const formatDate = (timestamp: number) => {
    return format(new Date(timestamp * 1000), "MMM d, yyyy");
  };

  const formatLastUsed = (timestamp: number | null) => {
    if (!timestamp) return "Never";
    return formatDistanceToNow(new Date(timestamp * 1000), { addSuffix: true });
  };

  const isExpired = (expiresAt: number | null) => {
    if (!expiresAt) return false;
    return expiresAt * 1000 < Date.now();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">API Keys</h3>
        <Button onClick={() => setCreateDialogOpen(true)}>Create API Key</Button>
      </div>

      {newKeyPlainText && (
        <Alert className="bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-900">
          <AlertDescription className="space-y-3">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-semibold text-green-900 dark:text-green-100">API key created successfully</p>
                <p className="text-sm text-green-800 dark:text-green-200 mt-1">
                  Make sure to copy this key now. You won't be able to see it again!
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={dismissSuccessBanner}
                className="h-6 w-6"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 p-3 bg-white dark:bg-gray-900 border border-green-300 dark:border-green-800 rounded font-mono text-sm break-all">
                {newKeyPlainText}
              </code>
              <Button onClick={handleCopyKey} variant="outline" size="sm">
                {copiedKey ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-16 bg-muted animate-pulse rounded" />
          ))}
        </div>
      ) : keys.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          No API keys yet. Create your first key to get started.
        </div>
      ) : (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-muted">
              <tr>
                <th className="text-left p-3 font-semibold text-sm">Name</th>
                <th className="text-left p-3 font-semibold text-sm">Permissions</th>
                <th className="text-left p-3 font-semibold text-sm">Created</th>
                <th className="text-left p-3 font-semibold text-sm">Expires</th>
                <th className="text-left p-3 font-semibold text-sm">Last Accessed</th>
                <th className="text-left p-3 font-semibold text-sm">Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <Fragment key={key.id}>
                  <tr
                    className="border-t hover:bg-muted/50 cursor-pointer"
                    onClick={() => handleRowClick(key)}
                  >
                    <td className="p-3">
                      <div className="flex items-center gap-2">
                        {expandedRowId === key.id ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                        {key.name}
                      </div>
                    </td>
                    <td className="p-3">
                      {key.permission_mode === PermissionMode.AllowAll ? (
                        <Badge variant="default">Full Access</Badge>
                      ) : (
                        <Badge variant="secondary">Limited Access</Badge>
                      )}
                    </td>
                    <td className="p-3 text-sm text-muted-foreground">{formatDate(key.created_at)}</td>
                    <td className="p-3">
                      {key.expires_at ? (
                        isExpired(key.expires_at) ? (
                          <Badge variant="destructive">Expired</Badge>
                        ) : (
                          <span className="text-sm text-muted-foreground">{formatDate(key.expires_at)}</span>
                        )
                      ) : (
                        <span className="text-sm text-muted-foreground">Never</span>
                      )}
                    </td>
                    <td className="p-3 text-sm text-muted-foreground">{formatLastUsed(key.last_used_at)}</td>
                    <td className="p-3">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteKey(key.id, key.name);
                        }}
                        title="Delete key"
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </td>
                  </tr>
                  {expandedRowId === key.id && (
                    <tr key={`${key.id}-expanded`} className="border-t bg-muted/30">
                      <td colSpan={6} className="p-4">
                        {key.permission_mode === PermissionMode.AllowAll ? (
                          <p className="text-sm text-muted-foreground">
                            This key has full access to all instances
                          </p>
                        ) : loadingPermissions[key.id] ? (
                          <p className="text-sm text-muted-foreground">Loading permissions...</p>
                        ) : permissions[key.id] ? (
                          <div className="space-y-2">
                            <p className="text-sm font-semibold">Allowed Instances:</p>
                            <ul className="text-sm space-y-1">
                              {permissions[key.id].map((perm) => (
                                <li key={perm.instance_id} className="flex items-center gap-2">
                                  <Check className="h-3 w-3 text-green-600" />
                                  {perm.instance_name}
                                </li>
                              ))}
                            </ul>
                          </div>
                        ) : (
                          <p className="text-sm text-muted-foreground">No permissions data</p>
                        )}
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <CreateApiKeyDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onKeyCreated={handleKeyCreated}
      />
    </div>
  );
}

export default ApiKeysSection;
