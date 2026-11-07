// Mirror of the server's allowlisted GET /api/v1/config response
// (ConfigResponse in pkg/server/handlers_system.go). Credential-bearing
// configuration is never part of this shape.

export interface DockerSummary {
  enabled: boolean
  image: string
}

export interface BackendSummary {
  command: string
  docker?: DockerSummary
}

export interface Backends {
  'llama-cpp': BackendSummary
  vllm: BackendSummary
  mlx: BackendSummary
  custom?: Record<string, BackendSummary>
}

export interface InstanceDefaults {
  idle_timeout: number
  auto_restart: boolean
  max_restarts: number
  restart_delay: number
  on_demand_start: boolean
}

export interface AppConfig {
  local_node?: string
  instance_defaults: InstanceDefaults
  backends: Backends
}
