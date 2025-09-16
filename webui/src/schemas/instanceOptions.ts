import { BackendType } from '@/types/instance'
import { z } from 'zod'

// Define the LlamaCpp backend options schema
export const LlamaCppBackendOptionsSchema = z.object({
  // Common params
  verbose_prompt: z.boolean().optional(),
  threads: z.number().optional(),
  threads_batch: z.number().optional(),
  cpu_mask: z.string().optional(),
  cpu_range: z.string().optional(),
  cpu_strict: z.number().optional(),
  prio: z.number().optional(),
  poll: z.number().optional(),
  cpu_mask_batch: z.string().optional(),
  cpu_range_batch: z.string().optional(),
  cpu_strict_batch: z.number().optional(),
  prio_batch: z.number().optional(),
  poll_batch: z.number().optional(),
  ctx_size: z.number().optional(),
  predict: z.number().optional(),
  batch_size: z.number().optional(),
  ubatch_size: z.number().optional(),
  keep: z.number().optional(),
  flash_attn: z.boolean().optional(),
  no_perf: z.boolean().optional(),
  escape: z.boolean().optional(),
  no_escape: z.boolean().optional(),
  rope_scaling: z.string().optional(),
  rope_scale: z.number().optional(),
  rope_freq_base: z.number().optional(),
  rope_freq_scale: z.number().optional(),
  yarn_orig_ctx: z.number().optional(),
  yarn_ext_factor: z.number().optional(),
  yarn_attn_factor: z.number().optional(),
  yarn_beta_slow: z.number().optional(),
  yarn_beta_fast: z.number().optional(),
  dump_kv_cache: z.boolean().optional(),
  no_kv_offload: z.boolean().optional(),
  cache_type_k: z.string().optional(),
  cache_type_v: z.string().optional(),
  defrag_thold: z.number().optional(),
  parallel: z.number().optional(),
  mlock: z.boolean().optional(),
  no_mmap: z.boolean().optional(),
  numa: z.string().optional(),
  device: z.string().optional(),
  override_tensor: z.array(z.string()).optional(),
  gpu_layers: z.number().optional(),
  split_mode: z.string().optional(),
  tensor_split: z.string().optional(),
  main_gpu: z.number().optional(),
  check_tensors: z.boolean().optional(),
  override_kv: z.array(z.string()).optional(),
  lora: z.array(z.string()).optional(),
  lora_scaled: z.array(z.string()).optional(),
  control_vector: z.array(z.string()).optional(),
  control_vector_scaled: z.array(z.string()).optional(),
  control_vector_layer_range: z.string().optional(),
  model: z.string().optional(),
  model_url: z.string().optional(),
  hf_repo: z.string().optional(),
  hf_repo_draft: z.string().optional(),
  hf_file: z.string().optional(),
  hf_repo_v: z.string().optional(),
  hf_file_v: z.string().optional(),
  hf_token: z.string().optional(),
  log_disable: z.boolean().optional(),
  log_file: z.string().optional(),
  log_colors: z.boolean().optional(),
  verbose: z.boolean().optional(),
  verbosity: z.number().optional(),
  log_prefix: z.boolean().optional(),
  log_timestamps: z.boolean().optional(),

  // Sampling params
  samplers: z.string().optional(),
  seed: z.number().optional(),
  sampling_seq: z.string().optional(),
  ignore_eos: z.boolean().optional(),
  temp: z.number().optional(),
  top_k: z.number().optional(),
  top_p: z.number().optional(),
  min_p: z.number().optional(),
  xtc_probability: z.number().optional(),
  xtc_threshold: z.number().optional(),
  typical: z.number().optional(),
  repeat_last_n: z.number().optional(),
  repeat_penalty: z.number().optional(),
  presence_penalty: z.number().optional(),
  frequency_penalty: z.number().optional(),
  dry_multiplier: z.number().optional(),
  dry_base: z.number().optional(),
  dry_allowed_length: z.number().optional(),
  dry_penalty_last_n: z.number().optional(),
  dry_sequence_breaker: z.array(z.string()).optional(),
  dynatemp_range: z.number().optional(),
  dynatemp_exp: z.number().optional(),
  mirostat: z.number().optional(),
  mirostat_lr: z.number().optional(),
  mirostat_ent: z.number().optional(),
  logit_bias: z.array(z.string()).optional(),
  grammar: z.string().optional(),
  grammar_file: z.string().optional(),
  json_schema: z.string().optional(),
  json_schema_file: z.string().optional(),

  // Example-specific params
  no_context_shift: z.boolean().optional(),
  special: z.boolean().optional(),
  no_warmup: z.boolean().optional(),
  spm_infill: z.boolean().optional(),
  pooling: z.string().optional(),
  cont_batching: z.boolean().optional(),
  no_cont_batching: z.boolean().optional(),
  mmproj: z.string().optional(),
  mmproj_url: z.string().optional(),
  no_mmproj: z.boolean().optional(),
  no_mmproj_offload: z.boolean().optional(),
  alias: z.string().optional(),
  host: z.string().optional(),
  port: z.number().optional(),
  path: z.string().optional(),
  no_webui: z.boolean().optional(),
  embedding: z.boolean().optional(),
  reranking: z.boolean().optional(),
  api_key: z.string().optional(),
  api_key_file: z.string().optional(),
  ssl_key_file: z.string().optional(),
  ssl_cert_file: z.string().optional(),
  chat_template_kwargs: z.string().optional(),
  timeout: z.number().optional(),
  threads_http: z.number().optional(),
  cache_reuse: z.number().optional(),
  metrics: z.boolean().optional(),
  slots: z.boolean().optional(),
  props: z.boolean().optional(),
  no_slots: z.boolean().optional(),
  slot_save_path: z.string().optional(),
  jinja: z.boolean().optional(),
  reasoning_format: z.string().optional(),
  reasoning_budget: z.number().optional(),
  chat_template: z.string().optional(),
  chat_template_file: z.string().optional(),
  no_prefill_assistant: z.boolean().optional(),
  slot_prompt_similarity: z.number().optional(),
  lora_init_without_apply: z.boolean().optional(),
  draft_max: z.number().optional(),
  draft_min: z.number().optional(),
  draft_p_min: z.number().optional(),
  ctx_size_draft: z.number().optional(),
  device_draft: z.string().optional(),
  gpu_layers_draft: z.number().optional(),
  model_draft: z.string().optional(),
  cache_type_k_draft: z.string().optional(),
  cache_type_v_draft: z.string().optional(),

  // Audio/TTS params
  model_vocoder: z.string().optional(),
  tts_use_guide_tokens: z.boolean().optional(),

  // Default model params
  embd_bge_small_en_default: z.boolean().optional(),
  embd_e5_small_en_default: z.boolean().optional(),
  embd_gte_small_default: z.boolean().optional(),
  fim_qwen_1_5b_default: z.boolean().optional(),
  fim_qwen_3b_default: z.boolean().optional(),
  fim_qwen_7b_default: z.boolean().optional(),
  fim_qwen_7b_spec: z.boolean().optional(),
  fim_qwen_14b_spec: z.boolean().optional(),
})

// Define the MLX backend options schema
export const MlxBackendOptionsSchema = z.object({
  // Basic connection options
  model: z.string().optional(),
  host: z.string().optional(),
  port: z.number().optional(),
  python_path: z.string().optional(),
  
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

// Backend options union
export const BackendOptionsSchema = z.union([
  LlamaCppBackendOptionsSchema,
  MlxBackendOptionsSchema,
])

// Define the main create instance options schema
export const CreateInstanceOptionsSchema = z.object({
  // Restart options
  auto_restart: z.boolean().optional(),
  max_restarts: z.number().optional(),
  restart_delay: z.number().optional(),
  idle_timeout: z.number().optional(),
  on_demand_start: z.boolean().optional(),

  // Backend configuration
  backend_type: z.enum([BackendType.LLAMA_CPP, BackendType.MLX_LM]).optional(),
  backend_options: BackendOptionsSchema.optional(),
})

// Infer the TypeScript types from the schemas
export type LlamaCppBackendOptions = z.infer<typeof LlamaCppBackendOptionsSchema>
export type MlxBackendOptions = z.infer<typeof MlxBackendOptionsSchema>
export type BackendOptions = z.infer<typeof BackendOptionsSchema>
export type CreateInstanceOptions = z.infer<typeof CreateInstanceOptionsSchema>

// Helper to get all field keys for CreateInstanceOptions
export function getAllFieldKeys(): (keyof CreateInstanceOptions)[] {
  return Object.keys(CreateInstanceOptionsSchema.shape) as (keyof CreateInstanceOptions)[]
}

// Helper to get all LlamaCpp backend option field keys
export function getAllLlamaCppFieldKeys(): (keyof LlamaCppBackendOptions)[] {
  return Object.keys(LlamaCppBackendOptionsSchema.shape) as (keyof LlamaCppBackendOptions)[]
}

// Helper to get all MLX backend option field keys
export function getAllMlxFieldKeys(): (keyof MlxBackendOptions)[] {
  return Object.keys(MlxBackendOptionsSchema.shape) as (keyof MlxBackendOptions)[]
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
  return 'text' // ZodString and others default to text
}

// Get field type for LlamaCpp backend options
export function getLlamaCppFieldType(key: keyof LlamaCppBackendOptions): 'text' | 'number' | 'boolean' | 'array' {
  const fieldSchema = LlamaCppBackendOptionsSchema.shape[key]
  if (!fieldSchema) return 'text'
  
  // Handle ZodOptional wrapper
  const innerSchema = fieldSchema instanceof z.ZodOptional ? fieldSchema.unwrap() : fieldSchema
  
  if (innerSchema instanceof z.ZodBoolean) return 'boolean'
  if (innerSchema instanceof z.ZodNumber) return 'number'
  if (innerSchema instanceof z.ZodArray) return 'array'
  return 'text' // ZodString and others default to text
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