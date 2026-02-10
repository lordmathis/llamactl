import { z } from 'zod'

// Define the LlamaCpp backend options schema
export const LlamaCppBackendOptionsSchema = z.object({
  // Common params (ordered as in llama-cpp.md)
  verbose_prompt: z.boolean().optional(),
  threads: z.number().optional(), // -t, --threads N
  threads_batch: z.number().optional(), // -tb, --threads-batch N
  cpu_mask: z.string().optional(), // -C, --cpu-mask M
  cpu_range: z.string().optional(), // -Cr, --cpu-range lo-hi
  cpu_strict: z.number().optional(), // --cpu-strict <0|1>
  prio: z.number().optional(), // --prio N
  poll: z.number().optional(), // --poll <0...100>
  cpu_mask_batch: z.string().optional(), // -Cb, --cpu-mask-batch M
  cpu_range_batch: z.string().optional(), // -Crb, --cpu-range-batch lo-hi
  cpu_strict_batch: z.number().optional(),
  prio_batch: z.number().optional(),
  poll_batch: z.number().optional(),
  ctx_size: z.number().optional(), // -c, --ctx-size N
  predict: z.number().optional(), // -n, --predict, --n-predict N
  batch_size: z.number().optional(), // -b, --batch-size N
  ubatch_size: z.number().optional(), // -ub, --ubatch-size N
  keep: z.number().optional(), // --keep N
  swa_full: z.boolean().optional(), // --swa-full
  flash_attn: z.string().optional(), // -fa, --flash-attn [on|off|auto]
  perf: z.boolean().optional(), // --perf
  no_perf: z.boolean().optional(), // --no-perf
  escape: z.boolean().optional(), // -e, --escape
  no_escape: z.boolean().optional(), // --no-escape
  rope_scaling: z.string().optional(), // --rope-scaling {none,linear,yarn}
  rope_scale: z.number().optional(), // --rope-scale N
  rope_freq_base: z.number().optional(),
  rope_freq_scale: z.number().optional(),
  yarn_orig_ctx: z.number().optional(),
  yarn_ext_factor: z.number().optional(),
  yarn_attn_factor: z.number().optional(),
  yarn_beta_slow: z.number().optional(),
  yarn_beta_fast: z.number().optional(),
  kv_offload: z.boolean().optional(), // -kvo, --kv-offload
  no_kv_offload: z.boolean().optional(), // -nkvo, --no-kv-offload
  repack: z.boolean().optional(), // --repack
  no_repack: z.boolean().optional(), // -nr, --no-repack
  no_host: z.boolean().optional(), // --no-host
  cache_type_k: z.string().optional(), // -ctk, --cache-type-k TYPE
  cache_type_v: z.string().optional(), // -ctv, --cache-type-v TYPE
  defrag_thold: z.number().optional(), // -dt, --defrag-thold N
  mlock: z.boolean().optional(), // --mlock
  mmap: z.boolean().optional(), // --mmap
  no_mmap: z.boolean().optional(), // --no-mmap
  direct_io: z.boolean().optional(), // -dio, --direct-io
  no_direct_io: z.boolean().optional(), // -ndio, --no-direct-io
  numa: z.string().optional(), // --numa TYPE
  device: z.string().optional(), // -dev, --device <dev1,dev2,..>
  override_tensor: z.array(z.string()).optional(), // -ot, --override-tensor
  cpu_moe: z.boolean().optional(), // -cmoe, --cpu-moe
  n_cpu_moe: z.number().optional(), // -ncmoe, --n-cpu-moe N
  gpu_layers: z.number().optional(), // -ngl, --gpu-layers, --n-gpu-layers N
  split_mode: z.string().optional(), // -sm, --split-mode {none,layer,row}
  tensor_split: z.string().optional(), // -ts, --tensor-split N0,N1,N2,...
  main_gpu: z.number().optional(), // -mg, --main-gpu INDEX
  fit: z.string().optional(), // -fit, --fit [on|off]
  fit_target: z.string().optional(), // -fitt, --fit-target MiB0,MiB1,MiB2,...
  fit_ctx: z.number().optional(), // -fitc, --fit-ctx N
  check_tensors: z.boolean().optional(), // --check-tensors
  override_kv: z.array(z.string()).optional(), // --override-kv KEY=TYPE:VALUE,...
  op_offload: z.boolean().optional(), // --op-offload
  no_op_offload: z.boolean().optional(), // --no-op-offload
  lora: z.array(z.string()).optional(), // --lora FNAME
  lora_scaled: z.array(z.string()).optional(), // --lora-scaled FNAME:SCALE,...
  control_vector: z.array(z.string()).optional(), // --control-vector FNAME
  control_vector_scaled: z.array(z.string()).optional(), // --control-vector-scaled FNAME:SCALE,...
  control_vector_layer_range: z.string().optional(), // --control-vector-layer-range START END
  model: z.string().optional(), // -m, --model FNAME
  model_url: z.string().optional(), // -mu, --model-url MODEL_URL
  docker_repo: z.string().optional(), // -dr, --docker-repo [<repo>/]<model>[:quant]
  hf_repo: z.string().optional(), // -hf, -hfr, --hf-repo <user>/<model>[:quant]
  hf_repo_draft: z.string().optional(), // -hfd, -hfrd, --hf-repo-draft <user>/<model>[:quant]
  hf_file: z.string().optional(), // -hff, --hf-file FILE
  hf_repo_v: z.string().optional(), // -hfv, -hfrv, --hf-repo-v <user>/<model>[:quant]
  hf_file_v: z.string().optional(), // -hffv, --hf-file-v FILE
  hf_token: z.string().optional(), // -hft, --hf-token TOKEN
  log_disable: z.boolean().optional(), // --log-disable
  log_file: z.string().optional(), // --log-file FNAME
  log_colors: z.string().optional(), // --log-colors [on|off|auto]
  verbose: z.boolean().optional(), // -v, --verbose, --log-verbose
  offline: z.boolean().optional(), // --offline
  verbosity: z.number().optional(), // -lv, --verbosity, --log-verbosity N
  log_prefix: z.boolean().optional(), // --log-prefix
  log_timestamps: z.boolean().optional(), // --log-timestamps
  cache_type_k_draft: z.string().optional(), // -ctkd, --cache-type-k-draft TYPE
  cache_type_v_draft: z.string().optional(), // -ctvd, --cache-type-v-draft TYPE

  // Sampling params (ordered as in llama-cpp.md)
  samplers: z.string().optional(), // --samplers SAMPLERS
  seed: z.number().optional(), // -s, --seed SEED
  sampling_seq: z.string().optional(), // --sampler-seq, --sampling-seq SEQUENCE
  ignore_eos: z.boolean().optional(), // --ignore-eos
  temp: z.number().optional(), // --temp N
  top_k: z.number().optional(), // --top-k N
  top_p: z.number().optional(), // --top-p N
  min_p: z.number().optional(), // --min-p N
  top_nsigma: z.number().optional(), // --top-nsigma N
  xtc_probability: z.number().optional(), // --xtc-probability N
  xtc_threshold: z.number().optional(), // --xtc-threshold N
  typical: z.number().optional(), // --typical N
  repeat_last_n: z.number().optional(), // --repeat-last-n N
  repeat_penalty: z.number().optional(), // --repeat-penalty N
  presence_penalty: z.number().optional(), // --presence-penalty N
  frequency_penalty: z.number().optional(), // --frequency-penalty N
  dry_multiplier: z.number().optional(), // --dry-multiplier N
  dry_base: z.number().optional(), // --dry-base N
  dry_allowed_length: z.number().optional(),
  dry_penalty_last_n: z.number().optional(),
  dry_sequence_breaker: z.array(z.string()).optional(), // --dry-sequence-breaker STRING
  adaptive_target: z.number().optional(), // --adaptive-target N
  adaptive_decay: z.number().optional(), // --adaptive-decay N
  dynatemp_range: z.number().optional(), // --dynatemp-range N
  dynatemp_exp: z.number().optional(), // --dynatemp-exp N
  mirostat: z.number().optional(), // --mirostat N
  mirostat_lr: z.number().optional(), // --mirostat-lr N
  mirostat_ent: z.number().optional(), // --mirostat-ent N
  logit_bias: z.array(z.string()).optional(), // -l, --logit-bias TOKEN_ID(+/-)BIAS
  grammar: z.string().optional(), // --grammar GRAMMAR
  grammar_file: z.string().optional(), // --grammar-file FNAME
  json_schema: z.string().optional(), // -j, --json-schema SCHEMA
  json_schema_file: z.string().optional(), // -jf, --json-schema-file FILE
  backend_sampling: z.boolean().optional(), // -bs, --backend-sampling

  // Server-specific params (ordered as in llama-cpp.md)
  ctx_checkpoints: z.number().optional(), // --ctx-checkpoints, --swa-checkpoints N
  cache_ram: z.number().optional(), // -cram, --cache-ram N
  lookup_cache_static: z.boolean().optional(), // -lcs, --lookup-cache-static
  lookup_cache_dynamic: z.boolean().optional(), // -lcd, --lookup-cache-dynamic
  kv_unified: z.boolean().optional(), // -kvu, --kv-unified
  no_kv_unified: z.boolean().optional(), // -no-kvu, --no-kv-unified
  context_shift: z.boolean().optional(), // --context-shift
  no_context_shift: z.boolean().optional(), // --no-context-shift
  reverse_prompt: z.string().optional(), // -r, --reverse-prompt PROMPT
  special: z.boolean().optional(), // -sp, --special
  warmup: z.boolean().optional(), // --warmup
  no_warmup: z.boolean().optional(), // --no-warmup
  spm_infill: z.boolean().optional(), // --spm-infill
  pooling: z.string().optional(), // --pooling {none,mean,cls,last,rank}
  parallel: z.number().optional(), // -np, --parallel N
  cont_batching: z.boolean().optional(), // -cb, --cont-batching
  no_cont_batching: z.boolean().optional(), // -nocb, --no-cont-batching
  mmproj: z.string().optional(), // -mm, --mmproj FILE
  mmproj_url: z.string().optional(), // -mmu, --mmproj-url URL
  mmproj_auto: z.boolean().optional(), // --mmproj-auto
  no_mmproj: z.boolean().optional(), // --no-mmproj
  no_mmproj_auto: z.boolean().optional(), // --no-mmproj-auto
  mmproj_offload: z.boolean().optional(), // --mmproj-offload
  no_mmproj_offload: z.boolean().optional(), // --no-mmproj-offload
  image_min_tokens: z.number().optional(), // --image-min-tokens N
  image_max_tokens: z.number().optional(), // --image-max-tokens N
  override_tensor_draft: z.array(z.string()).optional(), // -otd, --override-tensor-draft
  cpu_moe_draft: z.boolean().optional(), // -cmoed, --cpu-moe-draft
  n_cpu_moe_draft: z.number().optional(), // -ncmoed, --n-cpu-moe-draft N
  alias: z.string().optional(), // -a, --alias STRING
  host: z.string().optional(), // --host HOST
  port: z.number().optional(), // --port PORT
  path: z.string().optional(), // --path PATH
  api_prefix: z.string().optional(), // --api-prefix PREFIX
  webui_config: z.string().optional(), // --webui-config JSON
  webui_config_file: z.string().optional(), // --webui-config-file PATH
  webui: z.boolean().optional(), // --webui
  no_webui: z.boolean().optional(), // --no-webui
  embedding: z.boolean().optional(), // --embedding, --embeddings
  reranking: z.boolean().optional(), // --rerank, --reranking
  api_key: z.string().optional(), // --api-key KEY
  api_key_file: z.string().optional(), // --api-key-file FNAME
  ssl_key_file: z.string().optional(), // --ssl-key-file FNAME
  ssl_cert_file: z.string().optional(), // --ssl-cert-file FNAME
  chat_template_kwargs: z.string().optional(), // --chat-template-kwargs STRING
  timeout: z.number().optional(), // -to, --timeout N
  threads_http: z.number().optional(), // --threads-http N
  cache_prompt: z.boolean().optional(), // --cache-prompt
  no_cache_prompt: z.boolean().optional(), // --no-cache-prompt
  cache_reuse: z.number().optional(), // --cache-reuse N
  metrics: z.boolean().optional(), // --metrics
  props: z.boolean().optional(), // --props
  slots: z.boolean().optional(), // --slots
  no_slots: z.boolean().optional(), // --no-slots
  slot_save_path: z.string().optional(), // --slot-save-path PATH
  media_path: z.string().optional(), // --media-path PATH
  models_dir: z.string().optional(), // --models-dir PATH
  models_preset: z.string().optional(), // --models-preset PATH
  models_max: z.number().optional(), // --models-max N
  models_autoload: z.boolean().optional(), // --models-autoload
  no_models_autoload: z.boolean().optional(), // --no-models-autoload
  jinja: z.boolean().optional(), // --jinja
  no_jinja: z.boolean().optional(), // --no-jinja
  reasoning_format: z.string().optional(), // --reasoning-format FORMAT
  reasoning_budget: z.number().optional(), // --reasoning-budget N
  chat_template: z.string().optional(), // --chat-template JINJA_TEMPLATE
  chat_template_file: z.string().optional(), // --chat-template-file JINJA_TEMPLATE_FILE
  prefill_assistant: z.boolean().optional(), // --prefill-assistant
  no_prefill_assistant: z.boolean().optional(), // --no-prefill-assistant
  slot_prompt_similarity: z.number().optional(), // -sps, --slot-prompt-similarity SIMILARITY
  lora_init_without_apply: z.boolean().optional(),
  sleep_idle_seconds: z.number().optional(), // --sleep-idle-seconds SECONDS
  threads_draft: z.number().optional(), // -td, --threads-draft N
  threads_batch_draft: z.number().optional(), // -tbd, --threads-batch-draft N
  draft_max: z.number().optional(), // --draft, --draft-n, --draft-max N
  draft_min: z.number().optional(), // --draft-min, --draft-n-min N
  draft_p_min: z.number().optional(), // --draft-p-min P
  ctx_size_draft: z.number().optional(), // -cd, --ctx-size-draft N
  device_draft: z.string().optional(), // -devd, --device-draft <dev1,dev2,..>
  gpu_layers_draft: z.number().optional(), // -ngld, --gpu-layers-draft, --n-gpu-layers-draft N
  model_draft: z.string().optional(), // -md, --model-draft FNAME
  spec_replace: z.string().optional(), // --spec-replace TARGET DRAFT
  spec_type: z.string().optional(), // --spec-type TYPE
  spec_ngram_size_n: z.number().optional(), // --spec-ngram-size-n N
  spec_ngram_size_m: z.number().optional(), // --spec-ngram-size-m M
  spec_ngram_check_rate: z.number().optional(), // --spec-ngram-check-rate RATE
  spec_ngram_min_hits: z.number().optional(), // --spec-ngram-min-hits N
  model_vocoder: z.string().optional(), // -mv, --model-vocoder FNAME
  tts_use_guide_tokens: z.boolean().optional(),

  // Default model params (ordered as in llama-cpp.md)
  embd_gemma_default: z.boolean().optional(), // --embd-gemma-default
  fim_qwen_1b: z.boolean().optional(), // --fim-qwen-1
  fim_qwen_1_5b_default: z.boolean().optional(), // --fim-qwen-1.5b-default
  fim_qwen_3b_default: z.boolean().optional(), // --fim-qwen-3b-default
  fim_qwen_7b_default: z.boolean().optional(), // --fim-qwen-7b-default
  fim_qwen_7b_spec: z.boolean().optional(), // --fim-qwen-7b-spec
  fim_qwen_14b_spec: z.boolean().optional(), // --fim-qwen-14b-spec
  fim_qwen_30b_default: z.boolean().optional(), // --fim-qwen-30b-default
  gpt_oss_20b_default: z.boolean().optional(), // --gpt-oss-20b-default
  gpt_oss_120b_default: z.boolean().optional(), // --gpt-oss-120b-default
  vision_gemma_4b_default: z.boolean().optional(), // --vision-gemma-4b-default
  vision_gemma_12b_default: z.boolean().optional(), // --vision-gemma-12b-default
  extra_args: z.record(z.string(), z.string()).optional(),
})

// Infer the TypeScript type from the schema
export type LlamaCppBackendOptions = z.infer<typeof LlamaCppBackendOptionsSchema>

// Helper to get all LlamaCpp backend option field keys
export function getAllLlamaCppFieldKeys(): (keyof LlamaCppBackendOptions)[] {
  return Object.keys(LlamaCppBackendOptionsSchema.shape) as (keyof LlamaCppBackendOptions)[]
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
