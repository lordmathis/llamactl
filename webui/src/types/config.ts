export interface BackendSettings {
  command: string
  args: string[]
  environment?: Record<string, string>
  docker?: DockerSettings
  response_headers?: Record<string, string>
}

export interface DockerSettings {
  enabled: boolean
  image: string
  args: string[]
  environment?: Record<string, string>
}

export interface BackendConfig {
  'llama-cpp': BackendSettings
  vllm: BackendSettings
  mlx: BackendSettings
}

export interface ServerConfig {
  host: string
  port: number
  allowed_origins: string[]
  allowed_headers: string[]
  enable_swagger: boolean
  response_headers?: Record<string, string>
}

export interface InstancesConfig {
  port_range: [number, number]
  logs_dir: string
  auto_create_dirs: boolean
  max_instances: number
  max_running_instances: number
  enable_lru_eviction: boolean
  default_auto_restart: boolean
  default_max_restarts: number
  default_restart_delay: number
  default_on_demand_start: boolean
  on_demand_start_timeout: number
  timeout_check_interval: number
}

export interface DatabaseConfig {
  path: string
  max_open_connections: number
  max_idle_connections: number
  connection_max_lifetime: number
}

export interface AuthConfig {
  require_inference_auth: boolean
  require_management_auth: boolean
  management_keys: string[] // Will be empty in sanitized response
}

export interface NodeConfig {
  address: string
  api_key: string // Will be empty in sanitized response
}

export interface AppConfig {
  server: ServerConfig
  backends: BackendConfig
  instances: InstancesConfig
  database: DatabaseConfig
  auth: AuthConfig
  local_node: string
  nodes: Record<string, NodeConfig>
  data_dir: string
  version?: string
  commit_hash?: string
  build_time?: string
}
