import { z } from 'zod'

// Custom backend options reference a named backends.custom.<name> config
// entry. Args are literal strings; {port} and {model} are substituted by
// the server before launch.
export const CustomBackendOptionsSchema = z.object({
  name: z.string(),
  host: z.string().optional(),
  port: z.number().optional(),
  model: z.string().optional(),
  args: z.array(z.string()).optional(),
  health_path: z.string().optional(),
})

// Infer the TypeScript type from the schema
export type CustomBackendOptions = z.infer<typeof CustomBackendOptionsSchema>

// Helper to get all custom backend option field keys
export function getAllCustomFieldKeys(): (keyof CustomBackendOptions)[] {
  return Object.keys(CustomBackendOptionsSchema.shape) as (keyof CustomBackendOptions)[]
}

// Get field type for custom backend options
export function getCustomFieldType(key: keyof CustomBackendOptions): 'text' | 'number' | 'boolean' | 'array' {
  const fieldSchema = CustomBackendOptionsSchema.shape[key]
  if (!fieldSchema) return 'text'

  const innerSchema = fieldSchema instanceof z.ZodOptional ? fieldSchema.unwrap() : fieldSchema

  if (innerSchema instanceof z.ZodBoolean) return 'boolean'
  if (innerSchema instanceof z.ZodNumber) return 'number'
  if (innerSchema instanceof z.ZodArray) return 'array'
  return 'text'
}
