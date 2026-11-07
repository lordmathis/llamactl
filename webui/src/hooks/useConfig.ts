import { useConfig } from '@/contexts/ConfigContext'

// Helper hook to get instance default values from config
export const useInstanceDefaults = () => {
  const { config } = useConfig()

  if (!config?.instance_defaults) {
    return null
  }

  return {
    idleTimeout: config.instance_defaults.idle_timeout,
    autoRestart: config.instance_defaults.auto_restart,
    maxRestarts: config.instance_defaults.max_restarts,
    restartDelay: config.instance_defaults.restart_delay,
    onDemandStart: config.instance_defaults.on_demand_start,
  }
}

// Helper hook to get specific backend settings by backend type
export const useBackendSettings = (backendType: string | undefined) => {
  const { config } = useConfig()

  if (!config?.backends || !backendType) {
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

  // 'custom' never reaches here (backendKey is null for it); the narrowed
  // cast keeps the indexed access from unioning in backends.custom's type
  const backendConfig = config.backends[backendKey as 'llama-cpp' | 'vllm' | 'mlx']

  if (!backendConfig) {
    return null
  }

  return {
    command: backendConfig.command || '',
    dockerEnabled: backendConfig.docker?.enabled ?? false,
    dockerImage: backendConfig.docker?.image || '',
  }
}

// All configured custom backends (backends.custom), keyed by name
export const useCustomBackends = () => {
  const { config } = useConfig()

  return config?.backends?.custom ?? {}
}

// Get settings for one named custom backend (backends.custom.<name>)
export const useCustomBackendSettings = (name: string | undefined) => {
  const customBackends = useCustomBackends()

  if (!name) {
    return null
  }

  const settings = customBackends[name]

  if (!settings) {
    return null
  }

  return {
    command: settings.command || '',
    dockerEnabled: settings.docker?.enabled ?? false,
    dockerImage: settings.docker?.image || '',
  }
}
