// ui/src/components/nodes/NodesList.tsx

import { HardDrive } from 'lucide-react';
import { useEffect, useState } from 'react';
import NodeRow from '@/components/nodes/NodeRow';
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useConfig } from '@/contexts/ConfigContext';
import { type NodesMap, nodesApi } from '@/lib/api';

function NodesList() {
  const { config } = useConfig();
  const [nodes, setNodes] = useState<NodesMap | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadNodes = async () => {
      try {
        const data = await nodesApi.list();
        if (!cancelled) {
          setNodes(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : 'Failed to fetch nodes',
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadNodes();
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <div
        className="flex items-center justify-center py-12"
        role="status"
        aria-label="Loading"
      >
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading nodes...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-destructive mb-4">
          <p className="text-lg font-semibold">Error loading nodes</p>
          <p className="text-sm">{error}</p>
        </div>
      </div>
    );
  }

  const entries = nodes ? Object.entries(nodes) : [];
  const localNode = config?.local_node;

  entries.sort(([a], [b]) => {
    if (a === localNode && b !== localNode) return -1;
    if (b === localNode && a !== localNode) return 1;
    return a.localeCompare(b);
  });

  if (entries.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <HardDrive className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p className="text-lg mb-2">No nodes found</p>
        <p className="text-sm">
          Add nodes in the configuration file to get started
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-foreground mb-6">
        Nodes ({entries.length})
      </h2>

      <TooltipProvider>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Latency</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.map(([name, node]) => (
              <NodeRow
                key={name}
                name={name}
                address={node.address}
                isLocal={name === localNode}
              />
            ))}
          </TableBody>
        </Table>
      </TooltipProvider>
    </div>
  );
}

export default NodesList;
