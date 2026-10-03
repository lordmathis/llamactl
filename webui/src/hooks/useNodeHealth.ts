// ui/src/hooks/useNodeHealth.ts
import { useEffect, useState } from 'react';
import { nodeHealthService } from '@/lib/nodeHealthService';
import type { NodeHealth } from '@/types/node';

export function useNodeHealth(nodeName: string): NodeHealth | undefined {
  const [health, setHealth] = useState<NodeHealth | undefined>();

  useEffect(() => {
    return nodeHealthService.subscribe(nodeName, setHealth);
  }, [nodeName]);

  return health;
}
