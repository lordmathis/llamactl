import { z } from 'zod'

// Define the vLLM backend options schema
export const VllmBackendOptionsSchema = z.object({
  // Basic connection options (auto-assigned by llamactl)
  host: z.string().optional(),
  port: z.number().optional(),

  // Model and engine configuration
  model: z.string().optional(),
  tokenizer: z.string().optional(),
  skip_tokenizer_init: z.boolean().optional(),
  revision: z.string().optional(),
  code_revision: z.string().optional(),
  tokenizer_revision: z.string().optional(),
  tokenizer_mode: z.string().optional(),
  trust_remote_code: z.boolean().optional(),
  download_dir: z.string().optional(),
  load_format: z.string().optional(),
  config_format: z.string().optional(),
  dtype: z.string().optional(),
  kv_cache_dtype: z.string().optional(),
  quantization_param_path: z.string().optional(),
  seed: z.number().optional(),
  max_model_len: z.number().optional(),
  guided_decoding_backend: z.string().optional(),
  distributed_executor_backend: z.string().optional(),
  worker_use_ray: z.boolean().optional(),
  ray_workers_use_nsight: z.boolean().optional(),

  // Performance and serving configuration
  block_size: z.number().optional(),
  enable_prefix_caching: z.boolean().optional(),
  disable_sliding_window: z.boolean().optional(),
  use_v2_block_manager: z.boolean().optional(),
  num_lookahead_slots: z.number().optional(),
  swap_space: z.number().optional(),
  cpu_offload_gb: z.number().optional(),
  gpu_memory_utilization: z.number().optional(),
  num_gpu_blocks_override: z.number().optional(),
  max_num_batched_tokens: z.number().optional(),
  max_num_seqs: z.number().optional(),
  max_logprobs: z.number().optional(),
  disable_log_stats: z.boolean().optional(),
  quantization: z.string().optional(),
  rope_scaling: z.string().optional(),
  rope_theta: z.number().optional(),
  enforce_eager: z.boolean().optional(),
  max_context_len_to_capture: z.number().optional(),
  max_seq_len_to_capture: z.number().optional(),
  disable_custom_all_reduce: z.boolean().optional(),
  tokenizer_pool_size: z.number().optional(),
  tokenizer_pool_type: z.string().optional(),
  tokenizer_pool_extra_config: z.string().optional(),
  enable_lora_bias: z.boolean().optional(),
  lora_extra_vocab_size: z.number().optional(),
  lora_rank: z.number().optional(),
  prompt_lookback_distance: z.number().optional(),
  preemption_mode: z.string().optional(),

  // Distributed and parallel processing
  tensor_parallel_size: z.number().optional(),
  pipeline_parallel_size: z.number().optional(),
  max_parallel_loading_workers: z.number().optional(),
  disable_async_output_proc: z.boolean().optional(),
  worker_class: z.string().optional(),
  enabled_lora_modules: z.string().optional(),
  max_lora_rank: z.number().optional(),
  fully_sharded_loras: z.boolean().optional(),
  lora_modules: z.string().optional(),
  prompt_adapters: z.string().optional(),
  max_prompt_adapter_token: z.number().optional(),
  device: z.string().optional(),
  scheduler_delay: z.number().optional(),
  enable_chunked_prefill: z.boolean().optional(),
  speculative_model: z.string().optional(),
  speculative_model_quantization: z.string().optional(),
  speculative_revision: z.string().optional(),
  speculative_max_model_len: z.number().optional(),
  speculative_disable_by_batch_size: z.number().optional(),
  ngpt_speculative_length: z.number().optional(),
  speculative_disable_mqa: z.boolean().optional(),
  model_loader_extra_config: z.string().optional(),
  ignore_patterns: z.string().optional(),
  preloaded_lora_modules: z.string().optional(),

  // OpenAI server specific options
  uds: z.string().optional(),
  uvicorn_log_level: z.string().optional(),
  response_role: z.string().optional(),
  ssl_keyfile: z.string().optional(),
  ssl_certfile: z.string().optional(),
  ssl_ca_certs: z.string().optional(),
  ssl_cert_reqs: z.number().optional(),
  root_path: z.string().optional(),
  middleware: z.array(z.string()).optional(),
  return_tokens_as_token_ids: z.boolean().optional(),
  disable_frontend_multiprocessing: z.boolean().optional(),
  enable_auto_tool_choice: z.boolean().optional(),
  tool_call_parser: z.string().optional(),
  tool_server: z.string().optional(),
  chat_template: z.string().optional(),
  chat_template_content_format: z.string().optional(),
  allow_credentials: z.boolean().optional(),
  allowed_origins: z.array(z.string()).optional(),
  allowed_methods: z.array(z.string()).optional(),
  allowed_headers: z.array(z.string()).optional(),
  api_key: z.array(z.string()).optional(),
  enable_log_outputs: z.boolean().optional(),
  enable_token_usage: z.boolean().optional(),
  enable_async_engine_debug: z.boolean().optional(),
  engine_use_ray: z.boolean().optional(),
  disable_log_requests: z.boolean().optional(),
  max_log_len: z.number().optional(),

  // Additional engine configuration
  task: z.string().optional(),
  multi_modal_config: z.string().optional(),
  limit_mm_per_prompt: z.string().optional(),
  enable_sleep_mode: z.boolean().optional(),
  enable_chunking_request: z.boolean().optional(),
  compilation_config: z.string().optional(),
  disable_sliding_window_mask: z.boolean().optional(),
  enable_trtllm_engine_latency: z.boolean().optional(),
  override_pooling_config: z.string().optional(),
  override_neuron_config: z.string().optional(),
  override_kv_cache_align_size: z.number().optional(),
})

// Infer the TypeScript type from the schema
export type VllmBackendOptions = z.infer<typeof VllmBackendOptionsSchema>

// Helper to get all vLLM backend option field keys
export function getAllVllmFieldKeys(): (keyof VllmBackendOptions)[] {
  return Object.keys(VllmBackendOptionsSchema.shape) as (keyof VllmBackendOptions)[]
}

// Get field type for vLLM backend options
export function getVllmFieldType(key: keyof VllmBackendOptions): 'text' | 'number' | 'boolean' | 'array' {
  const fieldSchema = VllmBackendOptionsSchema.shape[key]
  if (!fieldSchema) return 'text'

  // Handle ZodOptional wrapper
  const innerSchema = fieldSchema instanceof z.ZodOptional ? fieldSchema.unwrap() : fieldSchema

  if (innerSchema instanceof z.ZodBoolean) return 'boolean'
  if (innerSchema instanceof z.ZodNumber) return 'number'
  if (innerSchema instanceof z.ZodArray) return 'array'
  return 'text' // ZodString and others default to text
}