import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { InstancesProvider, useInstances } from "@/contexts/InstancesContext";
import { instancesApi } from "@/lib/api";
import type { Instance } from "@/types/instance";
import { BackendType } from "@/types/instance";
import { AuthProvider } from "../AuthContext";

// Mock the API module
vi.mock("@/lib/api", () => ({
  instancesApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    restart: vi.fn(),
    delete: vi.fn(),
    getHealth: vi.fn(),
  },
}));

// Mock health service
vi.mock("@/lib/healthService", () => ({
  healthService: {
    subscribe: vi.fn(() => () => {}),
    refreshHealth: vi.fn(() => Promise.resolve()),
    checkHealthAfterOperation: vi.fn(),
    performHealthCheck: vi.fn(() => Promise.resolve({
      state: 'ready',
      instanceStatus: 'running',
      lastChecked: new Date(),
      source: 'http'
    })),
  },
  checkHealth: vi.fn(() => Promise.resolve({
    state: 'ready',
    instanceStatus: 'running',
    lastChecked: new Date(),
    source: 'http'
  })),
}));

// Test component to access context
function TestComponent() {
  const {
    instances,
    loading,
    error,
    createInstance,
    updateInstance,
    startInstance,
    stopInstance,
    restartInstance,
    deleteInstance,
    clearError,
  } = useInstances();

  return (
    <div>
      <div data-testid="loading">{loading.toString()}</div>
      <div data-testid="error">{error || "no-error"}</div>
      <div data-testid="instances-count">{instances.length}</div>
      {instances.map((instance) => (
        <div key={instance.name} data-testid={`instance-${instance.name}`}>
          {instance.name}:{instance.status}
        </div>
      ))}

      {/* Action buttons for testing with specific instances */}
      <button
        onClick={() => createInstance("new-instance", { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "test.gguf" } })}
        data-testid="create-instance"
      >
        Create Instance
      </button>
      <button
        onClick={() => updateInstance("instance1", { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "updated.gguf" } })}
        data-testid="update-instance"
      >
        Update Instance
      </button>
      <button
        onClick={() => startInstance("instance2")}
        data-testid="start-instance"
      >
        Start Instance2
      </button>
      <button
        onClick={() => stopInstance("instance1")}
        data-testid="stop-instance"
      >
        Stop Instance1
      </button>
      <button
        onClick={() => restartInstance("instance1")}
        data-testid="restart-instance"
      >
        Restart Instance1
      </button>
      <button
        onClick={() => deleteInstance("instance2")}
        data-testid="delete-instance"
      >
        Delete Instance2
      </button>
      <button onClick={clearError} data-testid="clear-error">
        Clear Error
      </button>
    </div>
  );
}

function renderWithProvider(children: ReactNode) {
  return render(
    <AuthProvider>
      <InstancesProvider>{children}</InstancesProvider>
    </AuthProvider>
  );
}

describe("InstancesContext", () => {
  const mockInstances: Instance[] = [
    { name: "instance1", status: "running", options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "model1.gguf" } } },
    { name: "instance2", status: "stopped", options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "model2.gguf" } } },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.setItem('llamactl_management_key', 'test-api-key-123');
    global.fetch = vi.fn(() => Promise.resolve(new Response(null, { status: 200 })));
    // Default successful API responses
    vi.mocked(instancesApi.list).mockResolvedValue(mockInstances);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("Initial Loading", () => {
    it("loads instances on mount", async () => {
      renderWithProvider(<TestComponent />);

      // Should start loading
      expect(screen.getByTestId("loading")).toHaveTextContent("true");

      // Should fetch instances
      await waitFor(() => {
        expect(instancesApi.list).toHaveBeenCalledOnce();
      });

      // Should display loaded instances
      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
        expect(screen.getByTestId("instance-instance1")).toHaveTextContent(
          "instance1:running"
        );
        expect(screen.getByTestId("instance-instance2")).toHaveTextContent(
          "instance2:stopped"
        );
      });
    });

    it("handles API error during initial load", async () => {
      const errorMessage = "Network error";
      vi.mocked(instancesApi.list).mockRejectedValue(new Error(errorMessage));

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("error")).toHaveTextContent(errorMessage);
        expect(screen.getByTestId("instances-count")).toHaveTextContent("0");
      });
    });
  });

  describe("Create Instance", () => {
    it("creates instance and adds it to state", async () => {
      const newInstance: Instance = {
        name: "new-instance",
        status: "stopped",
        options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "test.gguf" } },
      };
      vi.mocked(instancesApi.create).mockResolvedValue(newInstance);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      });

      screen.getByTestId("create-instance").click();

      await waitFor(() => {
        expect(instancesApi.create).toHaveBeenCalledWith("new-instance", {
          backend_type: BackendType.LLAMA_CPP,
          backend_options: { model: "test.gguf" }
        });
      });

      await waitFor(() => {
        expect(screen.getByTestId("instances-count")).toHaveTextContent("3");
        expect(screen.getByTestId("instance-new-instance")).toHaveTextContent(
          "new-instance:stopped"
        );
      });
    });

    it("handles create instance error without changing state", async () => {
      const errorMessage = "Instance already exists";
      vi.mocked(instancesApi.create).mockRejectedValue(new Error(errorMessage));

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      });

      screen.getByTestId("create-instance").click();

      await waitFor(() => {
        expect(screen.getByTestId("error")).toHaveTextContent(errorMessage);
      });

      expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      expect(
        screen.queryByTestId("instance-new-instance")
      ).not.toBeInTheDocument();
    });
  });

  describe("Update Instance", () => {
    it("updates instance and maintains it in state", async () => {
      const updatedInstance: Instance = {
        name: "instance1",
        status: "running",
        options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: "updated.gguf" } },
      };
      vi.mocked(instancesApi.update).mockResolvedValue(updatedInstance);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      });

      screen.getByTestId("update-instance").click();

      await waitFor(() => {
        expect(instancesApi.update).toHaveBeenCalledWith("instance1", {
          backend_type: BackendType.LLAMA_CPP,
          backend_options: { model: "updated.gguf" }
        });
      });

      await waitFor(() => {
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
        expect(screen.getByTestId("instance-instance1")).toBeInTheDocument();
      });
    });
  });

  describe("Start/Stop Instance", () => {
    it("starts existing instance and updates its running state", async () => {
      vi.mocked(instancesApi.start).mockResolvedValue({} as Instance);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        // instance2 starts as not running
        expect(screen.getByTestId("instance-instance2")).toHaveTextContent(
          "instance2:stopped"
        );
      });

      // Start instance2 (button already configured to start instance2)
      screen.getByTestId("start-instance").click();

      await waitFor(() => {
        expect(instancesApi.start).toHaveBeenCalledWith("instance2");
        // The running state should be updated to true
        expect(screen.getByTestId("instance-instance2")).toHaveTextContent(
          "instance2:running"
        );
      });
    });

    it("stops instance and updates running state to false", async () => {
      vi.mocked(instancesApi.stop).mockResolvedValue({} as Instance);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        // instance1 starts as running
        expect(screen.getByTestId("instance-instance1")).toHaveTextContent(
          "instance1:running"
        );
      });

      // Stop instance1 (button already configured to stop instance1)
      screen.getByTestId("stop-instance").click();

      await waitFor(() => {
        expect(instancesApi.stop).toHaveBeenCalledWith("instance1");
        // The running state should be updated to false
        expect(screen.getByTestId("instance-instance1")).toHaveTextContent(
          "instance1:stopped"
        );
      });
    });

    it("handles start instance error", async () => {
      const errorMessage = "Failed to start instance";
      vi.mocked(instancesApi.start).mockRejectedValue(new Error(errorMessage));

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
      });

      screen.getByTestId("start-instance").click();

      await waitFor(() => {
        expect(screen.getByTestId("error")).toHaveTextContent(errorMessage);
      });
    });
  });

  describe("Delete Instance", () => {
    it("deletes instance and removes it from state", async () => {
      vi.mocked(instancesApi.delete).mockResolvedValue(undefined);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
        expect(screen.getByTestId("instance-instance2")).toBeInTheDocument();
      });

      screen.getByTestId("delete-instance").click();

      await waitFor(() => {
        expect(instancesApi.delete).toHaveBeenCalledWith("instance2");
      });

      await waitFor(() => {
        expect(screen.getByTestId("instances-count")).toHaveTextContent("1");
        expect(
          screen.queryByTestId("instance-instance2")
        ).not.toBeInTheDocument();
        expect(screen.getByTestId("instance-instance1")).toBeInTheDocument(); // instance1 should still exist
      });
    });

    it("handles delete instance error without changing state", async () => {
      const errorMessage = "Instance is running";
      vi.mocked(instancesApi.delete).mockRejectedValue(new Error(errorMessage));

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      });

      screen.getByTestId("delete-instance").click();

      await waitFor(() => {
        expect(screen.getByTestId("error")).toHaveTextContent(errorMessage);
      });

      expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      expect(screen.getByTestId("instance-instance2")).toBeInTheDocument();
    });
  });

  describe("Error Management", () => {
    it("clears error when clearError is called", async () => {
      const errorMessage = "Test error";
      vi.mocked(instancesApi.list).mockRejectedValue(new Error(errorMessage));

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("error")).toHaveTextContent(errorMessage);
      });

      screen.getByTestId("clear-error").click();

      await waitFor(() => {
        expect(screen.getByTestId("error")).toHaveTextContent("no-error");
      });
    });
  });

  describe("State Consistency", () => {
    it("maintains consistent state during multiple operations", async () => {
      // Test that operations don't interfere with each other
      const newInstance: Instance = {
        name: "new-instance",
        status: "stopped",
        options: {},
      };
      vi.mocked(instancesApi.create).mockResolvedValue(newInstance);
      vi.mocked(instancesApi.start).mockResolvedValue({} as Instance);

      renderWithProvider(<TestComponent />);

      await waitFor(() => {
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("instances-count")).toHaveTextContent("2");
      });

      // Create new instance
      screen.getByTestId("create-instance").click();

      await waitFor(() => {
        expect(screen.getByTestId("instances-count")).toHaveTextContent("3");
      });

      // Start an instance (this should not affect the count)
      screen.getByTestId("start-instance").click();

      await waitFor(() => {
        expect(instancesApi.start).toHaveBeenCalled();
        expect(screen.getByTestId("instances-count")).toHaveTextContent("3"); // Still 3
        // But the running state should change
        expect(screen.getByTestId("instance-instance2")).toHaveTextContent(
          "instance2:running"
        );
      });
    });
  });
});
