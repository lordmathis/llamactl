import type { CreateInstanceOptions, Instance } from "@/types/instance";

const API_BASE = "/api/v1";

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

    // Handle authentication errors
    if (response.status === 401) {
      throw new Error('Authentication required');
    }

    if (!response.ok) {
      // Try to get error message from response
      let errorMessage = `HTTP ${response.status}`;
      try {
        const errorText = await response.text();
        if (errorText) {
          errorMessage += `: ${errorText}`;
        }
      } catch {
        // If we can't read the error, just use status
      }

      throw new Error(errorMessage);
    }

    // Handle empty responses (like DELETE)
    if (response.status === 204) {
      return undefined as T;
    }

    // Parse response based on type
    if (responseType === "text") {
      const text = await response.text();
      return text as T;
    } else {
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

// Server API functions
export const serverApi = {
  // GET /server/help
  getHelp: () => apiCall<string>("/server/help", {}, "text"),

  // GET /server/version
  getVersion: () => apiCall<string>("/server/version", {}, "text"),

  // GET /server/devices
  getDevices: () => apiCall<string>("/server/devices", {}, "text"),
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
  getHealth: (name: string) => apiCall<any>(`/instances/${name}/proxy/health`),
};
