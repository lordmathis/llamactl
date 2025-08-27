import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

export { type CreateInstanceOptions } from '@/schemas/instanceOptions'

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