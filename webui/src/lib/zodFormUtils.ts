import { CreateInstanceOptions, getAllFieldKeys } from '@/schemas/instanceOptions'

// Only define the basic fields we want to show by default
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

export function getBasicFields(): (keyof CreateInstanceOptions)[] {
  return Object.keys(basicFieldsConfig) as (keyof CreateInstanceOptions)[]
}

export function getAdvancedFields(): (keyof CreateInstanceOptions)[] {
  return getAllFieldKeys().filter(key => !isBasicField(key))
}

// Re-export the Zod-based functions
export { getFieldType } from '@/schemas/instanceOptions'