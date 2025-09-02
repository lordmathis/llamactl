import { type CreateInstanceOptions, type BackendOptions, getAllFieldKeys, getAllBackendFieldKeys } from '@/schemas/instanceOptions'

// Instance-level basic fields (not backend-specific)
export const basicFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  required?: boolean
}> = {
  auto_restart: {
    label: 'Auto Restart',
    description: 'Automatically restart the instance on failure'
  },
  max_restarts: {
    label: 'Max Restarts',
    placeholder: '3',
    description: 'Maximum number of restart attempts (0 = unlimited)'
  },
  restart_delay: {
    label: 'Restart Delay (seconds)',
    placeholder: '5',
    description: 'Delay in seconds before attempting restart'
  },
  idle_timeout: {
    label: 'Idle Timeout (minutes)',
    placeholder: '60',
    description: 'Time in minutes before instance is considered idle and stopped'
  },
  on_demand_start: {
    label: 'On-Demand Start',
    description: 'Start instance upon receiving OpenAI-compatible API request'
  },
  backend_type: {
    label: 'Backend Type',
    description: 'Type of backend to use for this instance'
  }
}

// Backend-specific basic fields (these go in backend_options)
export const basicBackendFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  required?: boolean
}> = {
  model: {
    label: 'Model Path',
    placeholder: '/path/to/model.gguf',
    description: 'Path to the model file'
  },
  hf_repo: {
    label: 'Hugging Face Repository',
    placeholder: 'microsoft/DialoGPT-medium',
    description: 'Hugging Face model repository'
  },
  hf_file: {
    label: 'Hugging Face File',
    placeholder: 'model.gguf',
    description: 'Specific file in the repository'
  },
  gpu_layers: {
    label: 'GPU Layers',
    placeholder: '0',
    description: 'Number of layers to offload to GPU'
  }
}

export function isBasicField(key: keyof CreateInstanceOptions): boolean {
  return key in basicFieldsConfig
}

export function isBasicBackendField(key: keyof BackendOptions): boolean {
  return key in basicBackendFieldsConfig
}

export function getBasicFields(): (keyof CreateInstanceOptions)[] {
  return Object.keys(basicFieldsConfig) as (keyof CreateInstanceOptions)[]
}

export function getAdvancedFields(): (keyof CreateInstanceOptions)[] {
  return getAllFieldKeys().filter(key => !isBasicField(key))
}

export function getBasicBackendFields(): (keyof BackendOptions)[] {
  return Object.keys(basicBackendFieldsConfig) as (keyof BackendOptions)[]
}

export function getAdvancedBackendFields(): (keyof BackendOptions)[] {
  return getAllBackendFieldKeys().filter(key => !isBasicBackendField(key))
}

// Re-export the Zod-based functions
export { getFieldType, getBackendFieldType } from '@/schemas/instanceOptions'