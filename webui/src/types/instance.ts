import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

export { type CreateInstanceOptions } from '@/schemas/instanceOptions'

export interface HealthStatus {
  status: 'ok' | 'loading' | 'error' | 'unknown'
  message?: string
  lastChecked: Date
}

export interface Instance {
  name: string;
  running: boolean;
  options?: CreateInstanceOptions;
}