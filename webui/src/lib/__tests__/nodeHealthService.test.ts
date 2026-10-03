import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { nodeHealthService } from '@/lib/nodeHealthService';

// Integration-style test: real service, real apiCall, mocked fetch returning
// the exact payloads the Go backend produces.
describe('nodeHealthService against real backend payloads', () => {
  const realPayloads: Record<string, string> = {
    '/api/v1/nodes/main/health':
      '{"status":"healthy","checked_at":1791062294}\n',
    '/api/v1/nodes/other/health':
      '{"status":"unreachable","error":"dial tcp: connection refused","checked_at":1791062294}\n',
  };

  beforeEach(() => {
    vi.restoreAllMocks();
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      for (const [path, body] of Object.entries(realPayloads)) {
        if (url.endsWith(path)) {
          return Promise.resolve(
            new Response(body, {
              status: 200,
              headers: { 'Content-Type': 'application/json' },
            }),
          );
        }
      }
      return Promise.resolve(new Response(null, { status: 404 }));
    }) as unknown as typeof fetch;
  });

  afterEach(() => {
    nodeHealthService.destroy();
  });

  it('maps a healthy backend response to a healthy state', async () => {
    const health = await nodeHealthService.performHealthCheck('main');
    expect(health.state).toBe('healthy');
    expect(health.error).toBeUndefined();
  });

  it('maps an unreachable backend response to an unreachable state with error', async () => {
    const health = await nodeHealthService.performHealthCheck('other');
    expect(health.state).toBe('unreachable');
    expect(health.error).toBe('dial tcp: connection refused');
  });

  it('reports unreachable when the endpoint 404s (stale backend without the route)', async () => {
    const health = await nodeHealthService.performHealthCheck('ghost');
    expect(health.state).toBe('unreachable');
    expect(health.error).toMatch(/404/);
  });
});
