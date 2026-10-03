import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import NodesList from '@/components/nodes/NodesList';
import { AuthProvider } from '@/contexts/AuthContext';
import { nodesApi } from '@/lib/api';
import { nodeHealthService } from '@/lib/nodeHealthService';
import type { NodeHealth } from '@/types/node';

vi.mock('@/lib/api', () => ({
  nodesApi: {
    list: vi.fn(),
    get: vi.fn(),
    getHealth: vi.fn(),
  },
  authApi: {
    whoami: vi.fn(() =>
      Promise.resolve({
        authenticated: false,
        oidc_enabled: false,
        user: null,
      }),
    ),
  },
}));

vi.mock('@/lib/nodeHealthService', () => ({
  nodeHealthService: {
    subscribe: vi.fn(() => () => {}),
  },
}));

vi.mock('@/contexts/ConfigContext', () => ({
  useConfig: () => ({
    config: { local_node: 'main' },
    isLoading: false,
    error: null,
  }),
}));

const mockNodes = {
  'worker-2': { address: 'http://worker-2:8080' },
  main: { address: '' },
  'worker-1': { address: 'http://worker-1:8080' },
};

const healthy: NodeHealth = {
  state: 'healthy',
  latencyMs: 12,
  lastChecked: new Date(),
};

function renderNodesList() {
  return render(
    <AuthProvider>
      <NodesList />
    </AuthProvider>,
  );
}

describe('NodesList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.setItem(
      'llamactl_management_key',
      'test-api-key-123',
    );
    global.fetch = vi.fn(() =>
      Promise.resolve(new Response(null, { status: 200 })),
    );
    // Radix tooltips need these in jsdom
    window.HTMLElement.prototype.hasPointerCapture = vi.fn(() => false);
    window.HTMLElement.prototype.setPointerCapture = vi.fn();
    window.HTMLElement.prototype.releasePointerCapture = vi.fn();
    class ResizeObserverMock {
      observe(): void {}
      unobserve(): void {}
      disconnect(): void {}
    }
    window.ResizeObserver =
      ResizeObserverMock as unknown as typeof ResizeObserver;

    vi.mocked(nodesApi.list).mockResolvedValue(mockNodes);
    vi.mocked(nodeHealthService.subscribe).mockImplementation(
      (_name: string, cb: (health: NodeHealth) => void) => {
        cb(healthy);
        return () => {};
      },
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows loading spinner while nodes are being fetched', () => {
    vi.mocked(nodesApi.list).mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve(mockNodes), 100)),
    );

    renderNodesList();

    expect(screen.getByText('Loading nodes...')).toBeInTheDocument();
  });

  it('displays error message when node loading fails', async () => {
    vi.mocked(nodesApi.list).mockRejectedValue(
      new Error('Failed to connect to server'),
    );

    renderNodesList();

    expect(await screen.findByText('Error loading nodes')).toBeInTheDocument();
    expect(screen.getByText('Failed to connect to server')).toBeInTheDocument();
  });

  it('shows empty state when no nodes exist', async () => {
    vi.mocked(nodesApi.list).mockResolvedValue({});

    renderNodesList();

    expect(await screen.findByText('No nodes found')).toBeInTheDocument();
    expect(screen.queryByText(/Nodes \(/)).not.toBeInTheDocument();
  });

  it('renders nodes with local node first and local/remote labels', async () => {
    renderNodesList();

    expect(await screen.findByText('Nodes (3)')).toBeInTheDocument();
    expect(screen.getByText('main')).toBeInTheDocument();
    expect(screen.getByText('worker-1')).toBeInTheDocument();
    expect(screen.getByText('worker-2')).toBeInTheDocument();
    expect(screen.getByText('Local')).toBeInTheDocument();
    expect(screen.getAllByText('Remote')).toHaveLength(2);

    const names = [
      screen.getByText('main'),
      screen.getByText('worker-1'),
      screen.getByText('worker-2'),
    ];
    expect(
      names[0].compareDocumentPosition(names[1]) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      names[1].compareDocumentPosition(names[2]) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it('shows the address as a tooltip when hovering a remote node', async () => {
    const user = userEvent.setup();
    renderNodesList();

    await screen.findByText('Nodes (3)');
    expect(screen.queryByText('http://worker-1:8080')).not.toBeInTheDocument();

    const remoteTypes = screen.getAllByTestId('node-remote');
    await user.hover(remoteTypes[0]);
    expect(await screen.findByTestId('node-address')).toHaveTextContent(
      'http://worker-1:8080',
    );
  });

  it('renders health status and latency per node', async () => {
    renderNodesList();

    expect(await screen.findAllByText('Healthy')).toHaveLength(3);
    expect(screen.getAllByText('12ms')).toHaveLength(2);
    expect(screen.getAllByText('—')).toHaveLength(1);
  });

  it('renders unreachable with error and no latency for a down node', async () => {
    vi.mocked(nodeHealthService.subscribe).mockImplementation(
      (name: string, cb: (health: NodeHealth) => void) => {
        cb(
          name === 'worker-1'
            ? {
                state: 'unreachable',
                error: 'connection refused',
                lastChecked: new Date(),
              }
            : healthy,
        );
        return () => {};
      },
    );

    renderNodesList();

    await screen.findByText('Nodes (3)');
    expect(screen.getByText('Unreachable')).toBeInTheDocument();
    expect(screen.getByTitle('connection refused')).toBeInTheDocument();
    expect(screen.getAllByText('12ms')).toHaveLength(1);
    expect(screen.getAllByText('—')).toHaveLength(2);
  });

  it('shows checking state while health is unknown', async () => {
    vi.mocked(nodeHealthService.subscribe).mockReturnValue(() => {});

    renderNodesList();

    expect(await screen.findAllByText('Checking')).toHaveLength(3);
  });
});
