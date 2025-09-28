import { BackendType } from '@/types/instance'
import { z } from 'zod'

// Import backend schemas from separate files
import {
  LlamaCppBackendOptionsSchema,
  type LlamaCppBackendOptions,
  getAllLlamaCppFieldKeys,
  getLlamaCppFieldType,
  MlxBackendOptionsSchema,
  type MlxBackendOptions,
  getAllMlxFieldKeys,
  getMlxFieldType,
  VllmBackendOptionsSchema,
  type VllmBackendOptions,
  getAllVllmFieldKeys,
  getVllmFieldType
} from './backends'

// Backend options union
export const BackendOptionsSchema = z.union([
  LlamaCppBackendOptionsSchema,
  MlxBackendOptionsSchema,
  VllmBackendOptionsSchema,
])

// Define the main create instance options schema
export const CreateInstanceOptionsSchema = z.object({
  // Restart options
  auto_restart: z.boolean().optional(),
  max_restarts: z.number().optional(),
  restart_delay: z.number().optional(),
  idle_timeout: z.number().optional(),
  on_demand_start: z.boolean().optional(),

  // Environment variables
  environment: z.record(z.string(), z.string()).optional(),

  // Backend configuration
  backend_type: z.enum([BackendType.LLAMA_CPP, BackendType.MLX_LM, BackendType.VLLM]).optional(),
  backend_options: BackendOptionsSchema.optional(),
})

// Re-export types and schemas from backend files
export {
  LlamaCppBackendOptionsSchema,
  MlxBackendOptionsSchema,
  VllmBackendOptionsSchema,
  type LlamaCppBackendOptions,
  type MlxBackendOptions,
  type VllmBackendOptions,
  getAllLlamaCppFieldKeys,
  getAllMlxFieldKeys,
  getAllVllmFieldKeys,
  getLlamaCppFieldType,
  getMlxFieldType,
  getVllmFieldType
}

// Infer the TypeScript types from the schemas
export type BackendOptions = z.infer<typeof BackendOptionsSchema>
export type CreateInstanceOptions = z.infer<typeof CreateInstanceOptionsSchema>

// Helper to get all field keys for CreateInstanceOptions
export function getAllFieldKeys(): (keyof CreateInstanceOptions)[] {
  return Object.keys(CreateInstanceOptionsSchema.shape) as (keyof CreateInstanceOptions)[]
}

// Get field type from Zod schema
export function getFieldType(key: keyof CreateInstanceOptions): 'text' | 'number' | 'boolean' | 'array' | 'object' {
  const fieldSchema = CreateInstanceOptionsSchema.shape[key]
  if (!fieldSchema) return 'text'

  // Handle ZodOptional wrapper
  const innerSchema = fieldSchema instanceof z.ZodOptional ? fieldSchema.unwrap() : fieldSchema

  if (innerSchema instanceof z.ZodBoolean) return 'boolean'
  if (innerSchema instanceof z.ZodNumber) return 'number'
  if (innerSchema instanceof z.ZodArray) return 'array'
  if (innerSchema instanceof z.ZodObject) return 'object'
  if (innerSchema instanceof z.ZodRecord) return 'object' // Handle ZodRecord as object
  return 'text' // ZodString and others default to text
}