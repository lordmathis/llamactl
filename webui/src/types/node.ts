export type NodeHealthState = 'healthy' | 'unreachable';

export interface NodeHealth {
  state: NodeHealthState;
  latencyMs?: number;
  error?: string;
  lastChecked: Date;
}
