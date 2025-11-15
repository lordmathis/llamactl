import { useConfig } from '@/contexts/ConfigContext'

// Helper hook to get instance default values from config
export const useInstanceDefaults = () => {
  const { config } = useConfig()

  if (!config || !config.instances) {
    return null
  }

  return {
    autoRestart: config.instances.default_auto_restart,
    maxRestarts: config.instances.default_max_restarts,
    restartDelay: config.instances.default_restart_delay,
    onDemandStart: config.instances.default_on_demand_start,
  }
}

// Helper hook to get specific backend settings by backend type
export const useBackendSettings = (backendType: string | undefined) => {
  const { config } = useConfig()

  if (!config || !config.backends || !backendType) {
    return null
  }

  // Map backend type to config key
  const backendKey = backendType === 'llama_cpp'
    ? 'llama-cpp'
    : backendType === 'mlx_lm'
    ? 'mlx'
    : backendType === 'vllm'
    ? 'vllm'
    : null

  if (!backendKey) {
    return null
  }

  const backendConfig = config.backends[backendKey as keyof typeof config.backends]

  if (!backendConfig) {
    return null
  }

  return {
    command: backendConfig.command || '',
    dockerEnabled: backendConfig.docker?.enabled ?? false,
    dockerImage: backendConfig.docker?.image || '',
  }
}
