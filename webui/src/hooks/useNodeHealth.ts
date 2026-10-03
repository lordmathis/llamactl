// ui/src/hooks/useNodeHealth.ts
import { useEffect, useState } from 'react';
import { nodeHealthService } from '@/lib/nodeHealthService';
import type { NodeHealth } from '@/types/node';

export function useNodeHealth(nodeName: string): NodeHealth | undefined {
  const [health, setHealth] = useState<NodeHealth | undefined>();

  useEffect(() => {
    // Subscribe to health updates for this node
    const unsubscribe = nodeHealthService.subscribe(
      nodeName,
      (healthStatus) => {
        setHealth(healthStatus);
      },
    );

    // Cleanup subscription on unmount or when node changes
    return unsubscribe;
  }, [nodeName]);

  return health;
}
