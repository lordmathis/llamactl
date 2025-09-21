import { z } from 'zod'

// Define the MLX backend options schema
export const MlxBackendOptionsSchema = z.object({
  // Basic connection options
  model: z.string().optional(),
  host: z.string().optional(),
  port: z.number().optional(),

  // Model and adapter options
  adapter_path: z.string().optional(),
  draft_model: z.string().optional(),
  num_draft_tokens: z.number().optional(),
  trust_remote_code: z.boolean().optional(),

  // Logging and templates
  log_level: z.enum(['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL']).optional(),
  chat_template: z.string().optional(),
  use_default_chat_template: z.boolean().optional(),
  chat_template_args: z.string().optional(), // JSON string

  // Sampling defaults
  temp: z.number().optional(),     // Note: MLX uses "temp" not "temperature"
  top_p: z.number().optional(),
  top_k: z.number().optional(),
  min_p: z.number().optional(),
  max_tokens: z.number().optional(),
})

// Infer the TypeScript type from the schema
export type MlxBackendOptions = z.infer<typeof MlxBackendOptionsSchema>

// Helper to get all MLX backend option field keys
export function getAllMlxFieldKeys(): (keyof MlxBackendOptions)[] {
  return Object.keys(MlxBackendOptionsSchema.shape) as (keyof MlxBackendOptions)[]
}

// Get field type for MLX backend options
export function getMlxFieldType(key: keyof MlxBackendOptions): 'text' | 'number' | 'boolean' | 'array' {
  const fieldSchema = MlxBackendOptionsSchema.shape[key]
  if (!fieldSchema) return 'text'

  // Handle ZodOptional wrapper
  const innerSchema = fieldSchema instanceof z.ZodOptional ? fieldSchema.unwrap() : fieldSchema

  if (innerSchema instanceof z.ZodBoolean) return 'boolean'
  if (innerSchema instanceof z.ZodNumber) return 'number'
  if (innerSchema instanceof z.ZodArray) return 'array'
  if (innerSchema instanceof z.ZodEnum) return 'text' // Enum treated as text/select
  return 'text' // ZodString and others default to text
}