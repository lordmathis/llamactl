import { nodesApi } from '@/lib/api';
import type { NodeHealth } from '@/types/node';

type NodeHealthCallback = (health: NodeHealth) => void;

const POLL_INTERVAL = 10000;
const CACHE_TTL = 2000; // dedupes concurrent checks

// Polls node health while the Nodes tab is mounted. First subscriber starts
// polling, last unsubscribe stops it.
class NodeHealthService {
  private intervals: Map<string, ReturnType<typeof setInterval>> = new Map();
  private callbacks: Map<string, Set<NodeHealthCallback>> = new Map();
  private healthCache: Map<string, { health: NodeHealth; timestamp: number }> =
    new Map();

  async performHealthCheck(nodeName: string): Promise<NodeHealth> {
    const cached = this.healthCache.get(nodeName);
    if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
      return cached.health;
    }

    let health: NodeHealth;
    try {
      const response = await nodesApi.getHealth(nodeName);
      health = {
        state: response.status,
        latencyMs: response.latency_ms,
        error: response.error,
        lastChecked: new Date(),
      };
    } catch (error) {
      // surface failures as a state, not an exception, so the UI can always render
      health = {
        state: 'unreachable',
        error: error instanceof Error ? error.message : 'Health check failed',
        lastChecked: new Date(),
      };
    }

    this.healthCache.set(nodeName, { health, timestamp: Date.now() });
    return health;
  }

  subscribe(nodeName: string, callback: NodeHealthCallback): () => void {
    if (!this.callbacks.has(nodeName)) {
      this.callbacks.set(nodeName, new Set());
    }

    const callbacks = this.callbacks.get(nodeName);
    if (callbacks) {
      callbacks.add(callback);

      if (callbacks.size === 1) {
        this.startHealthCheck(nodeName);
      }
    }

    return () => {
      const callbacks = this.callbacks.get(nodeName);
      if (callbacks) {
        callbacks.delete(callback);

        if (callbacks.size === 0) {
          this.stopHealthCheck(nodeName);
          this.callbacks.delete(nodeName);
          this.healthCache.delete(nodeName);
        }
      }
    };
  }

  private startHealthCheck(nodeName: string): void {
    if (this.intervals.has(nodeName)) {
      return; // already checking
    }

    // never replay a stale cached result on resubscribe
    this.healthCache.delete(nodeName);

    void this.refreshHealth(nodeName);

    const interval = setInterval(() => {
      void this.refreshHealth(nodeName);
    }, POLL_INTERVAL);

    this.intervals.set(nodeName, interval);
  }

  private async refreshHealth(nodeName: string): Promise<void> {
    try {
      const health = await this.performHealthCheck(nodeName);
      this.notifyCallbacks(nodeName, health);
    } catch (error) {
      // performHealthCheck catches API errors; this guards the unexpected
      console.error(`Node health check failed for ${nodeName}:`, error);
    }
  }

  private stopHealthCheck(nodeName: string): void {
    const interval = this.intervals.get(nodeName);
    if (interval) {
      clearInterval(interval);
      this.intervals.delete(nodeName);
    }
  }

  private notifyCallbacks(nodeName: string, health: NodeHealth): void {
    const callbacks = this.callbacks.get(nodeName);
    if (callbacks) {
      callbacks.forEach((callback) => {
        callback(health);
      });
    }
  }

  destroy(): void {
    this.intervals.forEach((interval) => {
      clearInterval(interval);
    });
    this.intervals.clear();
    this.callbacks.clear();
    this.healthCache.clear();
  }
}

export const nodeHealthService = new NodeHealthService();
