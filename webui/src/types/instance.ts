import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

export { type CreateInstanceOptions } from '@/schemas/instanceOptions'

export const BackendType = {
  LLAMA_CPP: 'llama_cpp',
  MLX_LM: 'mlx_lm',
  VLLM: 'vllm',
  // MLX_VLM: 'mlx_vlm',  // Future expansion
} as const

export type BackendTypeValue = typeof BackendType[keyof typeof BackendType]

export type InstanceStatus = 'running' | 'stopped' | 'failed' | 'restarting'

export type HealthState = 'stopped' | 'starting' | 'ready' | 'failed' | 'restarting'

export interface HealthStatus {
  state: HealthState
  instanceStatus: InstanceStatus
  lastChecked: Date
  error?: string
  source: 'backend' | 'http'
}

export interface Instance {
  name: string;
  status: InstanceStatus;
  options?: CreateInstanceOptions;
  docker_enabled?: boolean; // indicates backend is running via Docker
}