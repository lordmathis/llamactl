import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { nodeHealthService } from '@/lib/nodeHealthService';

// real service + real apiCall, only fetch is mocked
describe('nodeHealthService', () => {
  const okBody = (status: string) =>
    new Response(`{"status":"${status}","checked_at":1}\n`, {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });

  beforeEach(() => {
    global.fetch = vi.fn(() =>
      Promise.resolve(okBody('healthy')),
    ) as unknown as typeof fetch;
  });

  afterEach(() => {
    nodeHealthService.destroy();
  });

  it('maps a healthy response to a healthy state', async () => {
    const health = await nodeHealthService.performHealthCheck('main');
    expect(health.state).toBe('healthy');
    expect(health.error).toBeUndefined();
  });

  it('maps an unreachable response to an unreachable state with error', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve(
        new Response(
          '{"status":"unreachable","error":"connection refused","checked_at":1}\n',
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    ) as unknown as typeof fetch;

    const health = await nodeHealthService.performHealthCheck('other');
    expect(health.state).toBe('unreachable');
    expect(health.error).toBe('connection refused');
  });

  it('reports unreachable when the HTTP call fails', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve(new Response(null, { status: 404 })),
    ) as unknown as typeof fetch;

    const health = await nodeHealthService.performHealthCheck('ghost');
    expect(health.state).toBe('unreachable');
    expect(health.error).toMatch(/404/);
  });

  it('hits the network on resubscribe even if a failure was just cached', async () => {
    // subscribe against a dead backend, caching a failure
    global.fetch = vi.fn(() =>
      Promise.reject(new Error('backend down')),
    ) as unknown as typeof fetch;
    const unsubscribe = nodeHealthService.subscribe('main', () => {});
    await new Promise((r) => setTimeout(r, 10));
    unsubscribe();

    // resubscribe within the 2s cache TTL: must not replay the cached failure
    const fetchUrls: string[] = [];
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      fetchUrls.push(String(input));
      return Promise.resolve(okBody('healthy'));
    }) as unknown as typeof fetch;

    let received: { state: string } | undefined;
    const unsubscribe2 = nodeHealthService.subscribe('main', (h) => {
      received = h;
    });
    await new Promise((r) => setTimeout(r, 10));
    unsubscribe2();

    expect(fetchUrls.some((url) => url.includes('/nodes/main/health'))).toBe(
      true,
    );
    expect(received?.state).toBe('healthy');
  });
});
