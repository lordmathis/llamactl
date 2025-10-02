import type { CreateInstanceOptions, Instance } from "@/types/instance";
import { handleApiError } from "./errorUtils";

// Adding baseURI as a prefix to support being served behind a subpath
// e.g. when llmamctl's `/` is served behind a reverse proxy at `/proxy/...`
// the baseURI will be `/proxy/` and the API calls will be made to `/proxy/api/v1/<endpoint>`
export const API_BASE = document.baseURI + "api/v1";

// Base API call function with error handling
async function apiCall<T>(
  endpoint: string,
  options: RequestInit = {},
  responseType: "json" | "text" = "json"
): Promise<T> {
  const url = `${API_BASE}${endpoint}`;

  // Get auth token from sessionStorage (same as AuthContext)
  const storedKey = sessionStorage.getItem('llamactl_management_key');

  // Prepare headers with auth
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  // Add auth header if available
  if (storedKey) {
    headers['Authorization'] = `Bearer ${storedKey}`;
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    });

    // Handle errors using centralized error handler
    await handleApiError(response);

    // Handle empty responses (like DELETE)
    if (response.status === 204) {
      return undefined as T;
    }

    // Parse response based on type
    if (responseType === "text") {
      const text = await response.text();
      return text as T;
    } else {
      // Handle empty responses for JSON endpoints
      const contentLength = response.headers.get('content-length');
      if (contentLength === '0' || contentLength === null) {
        const text = await response.text();
        if (text.trim() === '') {
          return {} as T; // Return empty object for empty JSON responses
        }
      }
      const data = await response.json() as T;
      return data;
    }
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error("Network error occurred");
  }
}

// Server API functions (moved to llama-cpp backend)
export const serverApi = {
  // GET /backends/llama-cpp/help
  getHelp: () => apiCall<string>("/backends/llama-cpp/help", {}, "text"),

  // GET /backends/llama-cpp/version
  getVersion: () => apiCall<string>("/backends/llama-cpp/version", {}, "text"),

  // GET /backends/llama-cpp/devices
  getDevices: () => apiCall<string>("/backends/llama-cpp/devices", {}, "text"),
};

// Backend API functions
export const backendsApi = {
  llamaCpp: {
    // POST /backends/llama-cpp/parse-command
    parseCommand: (command: string) =>
      apiCall<CreateInstanceOptions>('/backends/llama-cpp/parse-command', {
        method: 'POST',
        body: JSON.stringify({ command }),
      }),
  },
  mlx: {
    // POST /backends/mlx/parse-command
    parseCommand: (command: string) =>
      apiCall<CreateInstanceOptions>('/backends/mlx/parse-command', {
        method: 'POST',
        body: JSON.stringify({ command }),
      }),
  },
  vllm: {
    // POST /backends/vllm/parse-command
    parseCommand: (command: string) =>
      apiCall<CreateInstanceOptions>('/backends/vllm/parse-command', {
        method: 'POST',
        body: JSON.stringify({ command }),
      }),
  },
};

// Node API types
export interface NodeResponse {
  name: string;
  address: string;
}

// Node API functions
export const nodesApi = {
  // GET /nodes
  list: () => apiCall<NodeResponse[]>("/nodes"),

  // GET /nodes/{name}
  get: (name: string) => apiCall<NodeResponse>(`/nodes/${name}`),
};

// Instance API functions
export const instancesApi = {
  // GET /instances
  list: () => apiCall<Instance[]>("/instances"),

  // GET /instances/{name}
  get: (name: string) => apiCall<Instance>(`/instances/${name}`),

  // POST /instances/{name}
  create: (name: string, options: CreateInstanceOptions) =>
    apiCall<Instance>(`/instances/${name}`, {
      method: "POST",
      body: JSON.stringify(options),
    }),

  // PUT /instances/{name}
  update: (name: string, options: CreateInstanceOptions) =>
    apiCall<Instance>(`/instances/${name}`, {
      method: "PUT",
      body: JSON.stringify(options),
    }),

  // DELETE /instances/{name}
  delete: (name: string) =>
    apiCall<void>(`/instances/${name}`, {
      method: "DELETE",
    }),

  // POST /instances/{name}/start
  start: (name: string) =>
    apiCall<Instance>(`/instances/${name}/start`, {
      method: "POST",
    }),

  // POST /instances/{name}/stop
  stop: (name: string) =>
    apiCall<Instance>(`/instances/${name}/stop`, {
      method: "POST",
    }),

  // POST /instances/{name}/restart
  restart: (name: string) =>
    apiCall<Instance>(`/instances/${name}/restart`, {
      method: "POST",
    }),

  // GET /instances/{name}/logs
  getLogs: (name: string, lines?: number) => {
    const params = lines ? `?lines=${lines}` : "";
    return apiCall<string>(`/instances/${name}/logs${params}`, {}, "text");
  },

  // GET /instances/{name}/proxy/health
  getHealth: (name: string) => apiCall<Record<string, unknown>>(`/instances/${name}/proxy/health`),
};
