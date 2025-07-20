export interface Instance {
  name: string;
  running: boolean;
  options?: CreateInstanceOptions;
}

export interface CreateInstanceOptions {

  auto_restart?: boolean;
  max_restarts?: number;
  restart_delay?: number;

  // Llama server options
  // Common params
  verbose_prompt?: boolean;
  threads?: number;
  threads_batch?: number;
  cpu_mask?: string;
  cpu_range?: string;
  cpu_strict?: number;
  priority?: number;
  poll?: number;
  cpu_mask_batch?: string;
  cpu_range_batch?: string;
  cpu_strict_batch?: number;
  priority_batch?: number;
  poll_batch?: number;
  ctx_size?: number;
  predict?: number;
  batch_size?: number;
  ubatch_size?: number;
  keep?: number;
  flash_attn?: boolean;
  no_perf?: boolean;
  escape?: boolean;
  no_escape?: boolean;
  rope_scaling?: string;
  rope_scale?: number;
  rope_freq_base?: number;
  rope_freq_scale?: number;
  yarn_orig_ctx?: number;
  yarn_ext_factor?: number;
  yarn_attn_factor?: number;
  yarn_beta_slow?: number;
  yarn_beta_fast?: number;
  dump_kv_cache?: boolean;
  no_kv_offload?: boolean;
  cache_type_k?: string;
  cache_type_v?: string;
  defrag_thold?: number;
  parallel?: number;
  mlock?: boolean;
  no_mmap?: boolean;
  numa?: string;
  device?: string;
  override_tensor?: string[];
  gpu_layers?: number;
  split_mode?: string;
  tensor_split?: string;
  main_gpu?: number;
  check_tensors?: boolean;
  override_kv?: string[];
  lora?: string[];
  lora_scaled?: string[];
  control_vector?: string[];
  control_vector_scaled?: string[];
  control_vector_layer_range?: string;
  model?: string;
  model_url?: string;
  hf_repo?: string;
  hf_repo_draft?: string;
  hf_file?: string;
  hf_repo_v?: string;
  hf_file_v?: string;
  hf_token?: string;
  log_disable?: boolean;
  log_file?: string;
  log_colors?: boolean;
  verbose?: boolean;
  verbosity?: number;
  log_prefix?: boolean;
  log_timestamps?: boolean;

  // Sampling params
  samplers?: string;
  seed?: number;
  sampling_seq?: string;
  ignore_eos?: boolean;
  temperature?: number;
  top_k?: number;
  top_p?: number;
  min_p?: number;
  xtc_probability?: number;
  xtc_threshold?: number;
  typical?: number;
  repeat_last_n?: number;
  repeat_penalty?: number;
  presence_penalty?: number;
  frequency_penalty?: number;
  dry_multiplier?: number;
  dry_base?: number;
  dry_allowed_length?: number;
  dry_penalty_last_n?: number;
  dry_sequence_breaker?: string[];
  dynatemp_range?: number;
  dynatemp_exp?: number;
  mirostat?: number;
  mirostat_lr?: number;
  mirostat_ent?: number;
  logit_bias?: string[];
  grammar?: string;
  grammar_file?: string;
  json_schema?: string;
  json_schema_file?: string;

  // Server/Example-specific params
  no_context_shift?: boolean;
  special?: boolean;
  no_warmup?: boolean;
  spm_infill?: boolean;
  pooling?: string;
  cont_batching?: boolean;
  no_cont_batching?: boolean;
  mmproj?: string;
  mmproj_url?: string;
  no_mmproj?: boolean;
  no_mmproj_offload?: boolean;
  alias?: string;
  host?: string;
  port?: number;
  path?: string;
  no_webui?: boolean;
  embedding?: boolean;
  reranking?: boolean;
  api_key?: string;
  api_key_file?: string;
  ssl_key_file?: string;
  ssl_cert_file?: string;
  chat_template_kwargs?: string;
  timeout?: number;
  threads_http?: number;
  cache_reuse?: number;
  metrics?: boolean;
  slots?: boolean;
  props?: boolean;
  no_slots?: boolean;
  slot_save_path?: string;
  jinja?: boolean;
  reasoning_format?: string;
  reasoning_budget?: number;
  chat_template?: string;
  chat_template_file?: string;
  no_prefill_assistant?: boolean;
  slot_prompt_similarity?: number;
  lora_init_without_apply?: boolean;

  // Speculative decoding params
  draft_max?: number;
  draft_min?: number;
  draft_p_min?: number;
  ctx_size_draft?: number;
  device_draft?: string;
  gpu_layers_draft?: number;
  model_draft?: string;
  cache_type_k_draft?: string;
  cache_type_v_draft?: string;

  // Audio/TTS params
  model_vocoder?: string;
  tts_use_guide_tokens?: boolean;

  // Default model params
  embd_bge_small_en_default?: boolean;
  embd_e5_small_en_default?: boolean;
  embd_gte_small_default?: boolean;
  fim_qwen_1_5b_default?: boolean;
  fim_qwen_3b_default?: boolean;
  fim_qwen_7b_default?: boolean;
  fim_qwen_7b_spec?: boolean;
  fim_qwen_14b_spec?: boolean;
}
