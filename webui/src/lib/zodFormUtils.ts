import {
  type LlamaCppBackendOptions,
  type MlxBackendOptions,
  type VllmBackendOptions,
  LlamaCppBackendOptionsSchema,
  MlxBackendOptionsSchema,
  VllmBackendOptionsSchema,
  getAllLlamaCppFieldKeys,
  getAllMlxFieldKeys,
  getAllVllmFieldKeys,
  getLlamaCppFieldType,
  getMlxFieldType,
  getVllmFieldType
} from '@/schemas/instanceOptions'

// LlamaCpp backend-specific basic fields
const basicLlamaCppFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  tooltip?: string
}> = {
  model: {
    label: 'Model Path',
    placeholder: '/models/gemma-3-1b-it-Q4_K_M.gguf',
    description: 'Local path to a .gguf model file. Leave empty to use HF Repo/File or router mode.',
    tooltip:
      'Absolute path to a locally stored GGUF model file. ' +
      'If you leave this empty and also leave HF Repo empty, llama-server starts in router mode ' +
      'and may auto-load any model presets found in your Hugging Face cache. ' +
      'Use the Preset tab to define router-mode models intentionally.'
  },
  hf_repo: {
    label: 'HF Repo',
    placeholder: 'ggml-org/gemma-3-1b-it-GGUF',
    description: 'Hugging Face repository to download a GGUF model from (e.g. ggml-org/gemma-3-1b-it-GGUF).',
    tooltip:
      'The Hugging Face repository ID in the form owner/repo-name. ' +
      'Must point to a repository that contains .gguf quantised model files. ' +
      'Combine with HF File to select a specific quantisation; if HF File is empty llama-server ' +
      'picks the first suitable file automatically.'
  },
  hf_file: {
    label: 'HF File',
    placeholder: 'gemma-3-1b-it-Q4_K_M.gguf',
    description: 'Specific .gguf filename inside the HF Repo (leave empty for auto-select).',
    tooltip:
      'The filename of the quantised model within the Hugging Face repository. ' +
      'For example gemma-3-1b-it-Q4_K_M.gguf. ' +
      'If left empty, llama-server selects the first .gguf file it finds in the repository, ' +
      'which may not be the quantisation level you want.'
  },
  gpu_layers: {
    label: 'GPU Layers',
    placeholder: '0',
    description: 'Layers to offload to GPU. Use -1 for all layers (full GPU), 0 for CPU only.',
    tooltip:
      'Controls how many transformer layers are offloaded to the GPU. ' +
      'Set to -1 to offload all layers for maximum GPU utilisation. ' +
      'Set to 0 to run entirely on CPU. ' +
      'Values between 1 and the model\'s total layer count allow partial offloading when VRAM is limited.'
  },
  models_preset: {
    label: 'Models Preset Path',
    placeholder: '/path/to/preset.ini',
    description: 'Path to a preset.ini that defines multiple models for router mode. Leave empty or use the Preset tab.',
    tooltip:
      'Router mode allows one llama-server instance to serve multiple models on demand. ' +
      'A preset.ini file defines each model\'s name and options. ' +
      'Leave this empty and use the Preset tab to have llamactl generate and manage the file automatically, ' +
      'or provide an absolute path to an existing preset.ini you maintain yourself.'
  }
}

// MLX backend-specific basic fields
const basicMlxFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
}> = {
  model: {
    label: 'Model',
    placeholder: 'mlx-community/Mistral-7B-Instruct-v0.3-4bit',
    description: 'The path to the MLX model weights, tokenizer, and config'
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

// vLLM backend-specific basic fields
const basicVllmFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
}> = {
  model: {
    label: 'Model',
    placeholder: 'microsoft/DialoGPT-medium',
    description: 'The name or path of the Hugging Face model to use'
  },
  tensor_parallel_size: {
    label: 'Tensor Parallel Size',
    placeholder: '1',
    description: 'Number of GPUs to use for distributed serving'
  },
  gpu_memory_utilization: {
    label: 'GPU Memory Utilization',
    placeholder: '0.9',
    description: 'The fraction of GPU memory to be used for the model executor'
  }
}

// Backend field configuration lookup
const backendFieldConfigs = {
  mlx_lm: basicMlxFieldsConfig,
  vllm: basicVllmFieldsConfig,
  llama_cpp: basicLlamaCppFieldsConfig,
} as const

const backendFieldGetters = {
  mlx_lm: getAllMlxFieldKeys,
  vllm: getAllVllmFieldKeys,
  llama_cpp: getAllLlamaCppFieldKeys,
} as const

export function getBasicBackendFields(backendType?: string): string[] {
  const normalizedType = (backendType || 'llama_cpp') as keyof typeof backendFieldConfigs
  const config = backendFieldConfigs[normalizedType] || basicLlamaCppFieldsConfig
  return Object.keys(config)
}

export function getAdvancedBackendFields(backendType?: string): string[] {
  const normalizedType = (backendType || 'llama_cpp') as keyof typeof backendFieldGetters
  const fieldGetter = backendFieldGetters[normalizedType] || getAllLlamaCppFieldKeys
  const basicConfig = backendFieldConfigs[normalizedType] || basicLlamaCppFieldsConfig

  return fieldGetter().filter(key => !(key in basicConfig) && key !== 'extra_args')
}

// Combined backend fields config for use in BackendFormField.
// llama.cpp is spread last so its per-field labels/descriptions take precedence
// over the generic vLLM/MLX ones for keys that appear in multiple backends (e.g. "model").
export const basicBackendFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
  tooltip?: string
}> = {
  ...basicVllmFieldsConfig,
  ...basicMlxFieldsConfig,
  ...basicLlamaCppFieldsConfig,
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

  // Try vLLM schema
  try {
    if (VllmBackendOptionsSchema.shape && key in VllmBackendOptionsSchema.shape) {
      return getVllmFieldType(key as keyof VllmBackendOptions)
    }
  } catch {
    // Schema might not be available
  }

  // Default fallback
  return 'text'
}

