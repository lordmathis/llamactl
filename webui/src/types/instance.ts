import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

export { type CreateInstanceOptions } from '@/schemas/instanceOptions'

export const BackendType = {
  LLAMA_CPP: 'llama_cpp',
  MLX_LM: 'mlx_lm',
  // MLX_VLM: 'mlx_vlm',  // Future expansion
} as const

export type BackendTypeValue = typeof BackendType[keyof typeof BackendType]

export type InstanceStatus = 'running' | 'stopped' | 'failed'

export interface HealthStatus {
  status: 'ok' | 'loading' | 'error' | 'unknown' | 'failed'
  message?: string
  lastChecked: Date
}

export interface Instance {
  name: string;
  status: InstanceStatus;
  options?: CreateInstanceOptions;
}