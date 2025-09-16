import { 
  type CreateInstanceOptions, 
  type LlamaCppBackendOptions, 
  type MlxBackendOptions,
  LlamaCppBackendOptionsSchema,
  MlxBackendOptionsSchema,
  getAllFieldKeys, 
  getAllLlamaCppFieldKeys,
  getAllMlxFieldKeys,
  getLlamaCppFieldType,
  getMlxFieldType
} from '@/schemas/instanceOptions'

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

// LlamaCpp backend-specific basic fields
const basicLlamaCppFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  required?: boolean
}> = {
  model: {
    label: 'Model Path',
    placeholder: '/path/to/model.gguf',
    description: 'Path to the model file',
    required: true
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

// MLX backend-specific basic fields
const basicMlxFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  required?: boolean
}> = {
  model: {
    label: 'Model',
    placeholder: 'mlx-community/Mistral-7B-Instruct-v0.3-4bit',
    description: 'The path to the MLX model weights, tokenizer, and config',
    required: true
  },
  python_path: {
    label: 'Python Virtual Environment Path',
    placeholder: '/path/to/venv',
    description: 'Path to Python virtual environment (optional)'
  },
  temp: {
    label: 'Temperature',
    placeholder: '0.0',
    description: 'Default sampling temperature (default: 0.0)'
  },
  top_p: {
    label: 'Top-P',
    placeholder: '1.0',
    description: 'Default nucleus sampling top-p (default: 1.0)'
  },
  top_k: {
    label: 'Top-K',
    placeholder: '0',
    description: 'Default top-k sampling (default: 0, disables top-k)'
  },
  min_p: {
    label: 'Min-P',
    placeholder: '0.0',
    description: 'Default min-p sampling (default: 0.0, disables min-p)'
  },
  max_tokens: {
    label: 'Max Tokens',
    placeholder: '512',
    description: 'Default maximum number of tokens to generate (default: 512)'
  }
}

function isBasicField(key: keyof CreateInstanceOptions): boolean {
  return key in basicFieldsConfig
}


export function getBasicFields(): (keyof CreateInstanceOptions)[] {
  return Object.keys(basicFieldsConfig) as (keyof CreateInstanceOptions)[]
}

export function getAdvancedFields(): (keyof CreateInstanceOptions)[] {
  return getAllFieldKeys().filter(key => !isBasicField(key))
}


export function getBasicBackendFields(backendType?: string): string[] {
  if (backendType === 'mlx_lm') {
    return Object.keys(basicMlxFieldsConfig)
  } else if (backendType === 'llama_cpp') {
    return Object.keys(basicLlamaCppFieldsConfig)
  }
  // Default to LlamaCpp for backward compatibility
  return Object.keys(basicLlamaCppFieldsConfig)
}

export function getAdvancedBackendFields(backendType?: string): string[] {
  if (backendType === 'mlx_lm') {
    return getAllMlxFieldKeys().filter(key => !(key in basicMlxFieldsConfig))
  } else if (backendType === 'llama_cpp') {
    return getAllLlamaCppFieldKeys().filter(key => !(key in basicLlamaCppFieldsConfig))
  }
  // Default to LlamaCpp for backward compatibility
  return getAllLlamaCppFieldKeys().filter(key => !(key in basicLlamaCppFieldsConfig))
}

// Combined backend fields config for use in BackendFormField
export const basicBackendFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  required?: boolean
}> = {
  ...basicLlamaCppFieldsConfig,
  ...basicMlxFieldsConfig
}

// Get field type for any backend option (union type)
export function getBackendFieldType(key: string): 'text' | 'number' | 'boolean' | 'array' {
  // Try to get type from LlamaCpp schema first
  try {
    if (LlamaCppBackendOptionsSchema.shape && key in LlamaCppBackendOptionsSchema.shape) {
      return getLlamaCppFieldType(key as keyof LlamaCppBackendOptions)
    }
  } catch {
    // Schema might not be available
  }
  
  // Try MLX schema
  try {
    if (MlxBackendOptionsSchema.shape && key in MlxBackendOptionsSchema.shape) {
      return getMlxFieldType(key as keyof MlxBackendOptions)
    }
  } catch {
    // Schema might not be available
  }
  
  // Default fallback
  return 'text'
}

// Re-export the Zod-based functions
export { getFieldType } from '@/schemas/instanceOptions'