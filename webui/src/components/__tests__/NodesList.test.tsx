import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import NodesList from '@/components/nodes/NodesList';
import { AuthProvider } from '@/contexts/AuthContext';
import { nodesApi } from '@/lib/api';
import { nodeHealthService } from '@/lib/nodeHealthService';
import type { NodeHealth } from '@/types/node';

// Mock the API
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

// Mock node health service to avoid real network calls and intervals
vi.mock('@/lib/nodeHealthService', () => ({
  nodeHealthService: {
    subscribe: vi.fn(() => () => {}),
  },
}));

// Mock config context so NodesList can identify the local node
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

function renderNodesList() {
  return render(
    <AuthProvider>
      <NodesList />
    </AuthProvider>,
  );
}

describe('NodesList', () => {
  const healthy: NodeHealth = {
    state: 'healthy',
    latencyMs: 12,
    lastChecked: new Date(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.setItem(
      'llamactl_management_key',
      'test-api-key-123',
    );
    global.fetch = vi.fn(() =>
      Promise.resolve(new Response(null, { status: 200 })),
    );
    // jsdom lacks the pointer capture and ResizeObserver APIs that Radix
    // tooltips use when opening on hover
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
    // Deliver health synchronously when a node subscribes
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

  describe('Loading State', () => {
    it('shows loading spinner while nodes are being fetched', () => {
      vi.mocked(nodesApi.list).mockImplementation(
        () =>
          new Promise((resolve) => setTimeout(() => resolve(mockNodes), 100)),
      );

      renderNodesList();

      expect(screen.getByText('Loading nodes...')).toBeInTheDocument();
      expect(screen.getByLabelText('Loading')).toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('displays error message when node loading fails', async () => {
      vi.mocked(nodesApi.list).mockRejectedValue(
        new Error('Failed to connect to server'),
      );

      renderNodesList();

      expect(
        await screen.findByText('Error loading nodes'),
      ).toBeInTheDocument();
      expect(
        screen.getByText('Failed to connect to server'),
      ).toBeInTheDocument();
    });
  });

  describe('Empty State', () => {
    it('shows empty state message when no nodes exist', async () => {
      vi.mocked(nodesApi.list).mockResolvedValue({});

      renderNodesList();

      expect(await screen.findByText('No nodes found')).toBeInTheDocument();
      expect(screen.queryByText(/Nodes \(/)).not.toBeInTheDocument();
    });
  });

  describe('Nodes Display', () => {
    it('displays all nodes with correct count', async () => {
      renderNodesList();

      expect(await screen.findByText('Nodes (3)')).toBeInTheDocument();
      expect(screen.getByText('main')).toBeInTheDocument();
      expect(screen.getByText('worker-1')).toBeInTheDocument();
      expect(screen.getByText('worker-2')).toBeInTheDocument();
    });

    it('shows the local node first, then remote nodes alphabetically', async () => {
      renderNodesList();

      await screen.findByText('Nodes (3)');

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

    it('labels the local node and remote nodes', async () => {
      renderNodesList();

      await screen.findByText('Nodes (3)');

      expect(screen.getByText('Local')).toBeInTheDocument();
      expect(screen.getAllByText('Remote')).toHaveLength(2);
    });

    it('shows the address as a tooltip when hovering a remote node', async () => {
      const user = userEvent.setup();
      renderNodesList();

      await screen.findByText('Nodes (3)');

      // Addresses are not rendered until the remote type text is hovered
      expect(
        screen.queryByText('http://worker-1:8080'),
      ).not.toBeInTheDocument();

      const remoteTypes = screen.getAllByTestId('node-remote');
      expect(remoteTypes).toHaveLength(2);

      // The first remote row is worker-1 (sorted alphabetically after main)
      await user.hover(remoteTypes[0]);
      expect(await screen.findByTestId('node-address')).toHaveTextContent(
        'http://worker-1:8080',
      );
    });
  });

  describe('Node Health', () => {
    it('subscribes each node to the health service', async () => {
      renderNodesList();

      await screen.findByText('Nodes (3)');

      expect(nodeHealthService.subscribe).toHaveBeenCalledWith(
        'main',
        expect.any(Function),
      );
      expect(nodeHealthService.subscribe).toHaveBeenCalledWith(
        'worker-1',
        expect.any(Function),
      );
      expect(nodeHealthService.subscribe).toHaveBeenCalledWith(
        'worker-2',
        expect.any(Function),
      );
    });

    it('renders health status and latency in separate columns', async () => {
      renderNodesList();

      expect(await screen.findAllByText('Healthy')).toHaveLength(3);
      // Only remote nodes show latency; the local node is never pinged
      expect(screen.getAllByText('12ms')).toHaveLength(2);
      expect(screen.getAllByText('—')).toHaveLength(1);
      expect(screen.getByText('Latency')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
    });

    it('shows no latency for unreachable nodes', async () => {
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
      expect(screen.getAllByText(/Healthy/)).toHaveLength(2);
      // Unreachable node and local node show no latency; one healthy remote does
      expect(screen.getAllByText('12ms')).toHaveLength(1);
      expect(screen.getAllByText('—')).toHaveLength(2);
    });

    it('shows checking state while health is unknown', async () => {
      vi.mocked(nodeHealthService.subscribe).mockReturnValue(() => {});

      renderNodesList();

      expect(await screen.findAllByText('Checking')).toHaveLength(3);
    });
  });
});
