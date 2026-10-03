import { nodesApi } from '@/lib/api';
import type { NodeHealth } from '@/types/node';

type NodeHealthCallback = (health: NodeHealth) => void;

// Polling interval for node health (in milliseconds)
const POLL_INTERVAL = 10000;
// Short cache to dedupe concurrent checks
const CACHE_TTL = 2000;

/**
 * Singleton service that polls node health while the Nodes tab is active.
 * Follows the same subscribe/notify lifecycle as healthService: the first
 * subscriber starts an immediate check plus an interval, the last
 * unsubscribe stops polling and clears cached state.
 */
class NodeHealthService {
  private intervals: Map<string, ReturnType<typeof setInterval>> = new Map();
  private callbacks: Map<string, Set<NodeHealthCallback>> = new Map();
  private healthCache: Map<string, { health: NodeHealth; timestamp: number }> =
    new Map();

  async performHealthCheck(nodeName: string): Promise<NodeHealth> {
    // Check cache first
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
      // Backend unreachable or node unknown - surface as unreachable state
      // rather than an exception so callers can always render a badge
      health = {
        state: 'unreachable',
        error: error instanceof Error ? error.message : 'Health check failed',
        lastChecked: new Date(),
      };
    }

    this.healthCache.set(nodeName, { health, timestamp: Date.now() });
    return health;
  }

  /**
   * Subscribe to health updates for a node. The returned function
   * unsubscribes and stops polling when the last subscriber leaves.
   */
  subscribe(nodeName: string, callback: NodeHealthCallback): () => void {
    if (!this.callbacks.has(nodeName)) {
      this.callbacks.set(nodeName, new Set());
    }

    const callbacks = this.callbacks.get(nodeName);
    if (callbacks) {
      callbacks.add(callback);

      // Start health checking if this is the first subscriber
      if (callbacks.size === 1) {
        this.startHealthCheck(nodeName);
      }
    }

    return () => {
      const callbacks = this.callbacks.get(nodeName);
      if (callbacks) {
        callbacks.delete(callback);

        // Stop health checking if no more subscribers
        if (callbacks.size === 0) {
          this.stopHealthCheck(nodeName);
          this.callbacks.delete(nodeName);
          this.healthCache.delete(nodeName);
        }
      }
    };
  }

  /**
   * Start polling a node: immediate check plus a fixed interval
   */
  private startHealthCheck(nodeName: string): void {
    if (this.intervals.has(nodeName)) {
      return; // Already checking
    }

    // Initial check immediately
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
      // performHealthCheck already converts API errors into health states,
      // so this only guards against unexpected failures
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

  /**
   * Stop all health checking and cleanup
   */
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
