import type { CreateInstanceOptions, Instance } from "@/types/instance";
import type { AppConfig } from "@/types/config";
import type { ApiKey, CreateKeyRequest, CreateKeyResponse, KeyPermissionResponse } from "@/types/apiKey";
import type { DownloadJob, CachedModel } from "@/types/model";
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
      if (contentLength === '0') {
        return {} as T; // Return empty object for empty JSON responses
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

  // GET /config
  getConfig: () => apiCall<AppConfig>("/config"),
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
  address: string;
}

export type NodesMap = Record<string, NodeResponse>;

// Node API functions
export const nodesApi = {
  // GET /nodes - returns map of node name to NodeResponse
  list: () => apiCall<NodesMap>("/nodes"),

  // GET /nodes/{name}
  get: (name: string) => apiCall<NodeResponse>(`/nodes/${encodeURIComponent(name)}`),
};

// Instance API functions
export const instancesApi = {
  // GET /instances
  list: () => apiCall<Instance[]>("/instances"),

  // GET /instances/{name}
  get: (name: string) => apiCall<Instance>(`/instances/${encodeURIComponent(name)}`),

  // POST /instances/{name}
  create: (name: string, options: CreateInstanceOptions) =>
    apiCall<Instance>(`/instances/${encodeURIComponent(name)}`, {
      method: "POST",
      body: JSON.stringify(options),
    }),

  // PUT /instances/{name}
  update: (name: string, options: CreateInstanceOptions) =>
    apiCall<Instance>(`/instances/${encodeURIComponent(name)}`, {
      method: "PUT",
      body: JSON.stringify(options),
    }),

  // DELETE /instances/{name}
  delete: (name: string) =>
    apiCall<void>(`/instances/${encodeURIComponent(name)}`, {
      method: "DELETE",
    }),

  // POST /instances/{name}/start
  start: (name: string) =>
    apiCall<Instance>(`/instances/${encodeURIComponent(name)}/start`, {
      method: "POST",
    }),

  // POST /instances/{name}/stop
  stop: (name: string) =>
    apiCall<Instance>(`/instances/${encodeURIComponent(name)}/stop`, {
      method: "POST",
    }),

  // POST /instances/{name}/restart
  restart: (name: string) =>
    apiCall<Instance>(`/instances/${encodeURIComponent(name)}/restart`, {
      method: "POST",
    }),

  // GET /instances/{name}/logs
  getLogs: (name: string, lines?: number) => {
    const params = lines ? `?lines=${lines}` : "";
    return apiCall<string>(`/instances/${encodeURIComponent(name)}/logs${params}`, {}, "text");
  },

  // GET /instances/{name}/proxy/health
  getHealth: (name: string) => apiCall<Record<string, unknown>>(`/instances/${encodeURIComponent(name)}/proxy/health`),
};

// API Keys API functions
export const apiKeysApi = {
  // GET /auth/keys
  list: () => apiCall<ApiKey[]>("/auth/keys"),

  // GET /auth/keys/{id}
  get: (id: number) => apiCall<ApiKey>(`/auth/keys/${id}`),

  // POST /auth/keys
  create: (request: CreateKeyRequest) =>
    apiCall<CreateKeyResponse>("/auth/keys", {
      method: "POST",
      body: JSON.stringify(request),
    }),

  // DELETE /auth/keys/{id}
  delete: (id: number) =>
    apiCall<void>(`/auth/keys/${id}`, {
      method: "DELETE",
    }),

  // GET /auth/keys/{id}/permissions
  getPermissions: (id: number) =>
    apiCall<KeyPermissionResponse[]>(`/auth/keys/${id}/permissions`),
};

// Llama.cpp model management types
export interface Model {
  id: string;
  object: string;
  owned_by: string;
  created: number;
  in_cache: boolean;
  path: string;
  status?: {
    value: string; // "loaded" | "loading" | "unloaded"
    args: string[];
  };
}

export interface ModelsListResponse {
  object: string;
  data: Model[];
}

// Llama.cpp model management API functions
export const llamaCppApi = {
  // GET /llama-cpp/{name}/models
  getModels: async (instanceName: string): Promise<Model[]> => {
    const response = await apiCall<ModelsListResponse>(
      `/llama-cpp/${encodeURIComponent(instanceName)}/models`
    );
    return response.data;
  },

  // POST /llama-cpp/{name}/models/{model}/load
  loadModel: (instanceName: string, modelName: string) =>
    apiCall<{ success: boolean }>(
      `/llama-cpp/${encodeURIComponent(instanceName)}/models/${encodeURIComponent(modelName)}/load`,
      {
        method: "POST",
        body: JSON.stringify({ model: modelName }),
      }
    ),

  // POST /llama-cpp/{name}/models/{model}/unload
  unloadModel: (instanceName: string, modelName: string) =>
    apiCall<{ success: boolean }>(
      `/llama-cpp/${encodeURIComponent(instanceName)}/models/${encodeURIComponent(modelName)}/unload`,
      {
        method: "POST",
        body: JSON.stringify({ model: modelName }),
      }
    ),
};

// Llama.cpp models cache management API functions
export const llamaCppModelsApi = {
  // Download management
  startDownload: (repo: string, tag?: string) =>
    apiCall<{ job_id: string; repo: string; tag: string }>(
      '/backends/llama-cpp/models/download',
      {
        method: 'POST',
        body: JSON.stringify({ repo, tag: tag || 'latest' })
      }
    ),

  getJob: (jobId: string) =>
    apiCall<DownloadJob>(`/backends/llama-cpp/models/jobs/${jobId}`),

  listJobs: () =>
    apiCall<{ jobs: DownloadJob[] }>('/backends/llama-cpp/models/jobs'),

  cancelJob: (jobId: string) =>
    apiCall<void>(`/backends/llama-cpp/models/jobs/${jobId}`, {
      method: 'DELETE'
    }),

  // Cache management
  listModels: () =>
    apiCall<CachedModel[]>('/backends/llama-cpp/models'),

  deleteModel: (repo: string, tag?: string) => {
    const params = new URLSearchParams({ repo })
    if (tag) params.append('tag', tag)
    return apiCall<void>(`/backends/llama-cpp/models?${params}`, {
      method: 'DELETE'
    })
  }
};
