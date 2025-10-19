package backends

// vllmMultiValuedFlags defines flags that should be repeated for each value rather than comma-separated
var vllmMultiValuedFlags = map[string]bool{
	"api-key":         true,
	"allowed-origins": true,
	"allowed-methods": true,
	"allowed-headers": true,
	"middleware":      true,
}

type VllmServerOptions struct {
	// Basic connection options (auto-assigned by llamactl)
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`

	// Model and engine configuration
	Model                      string `json:"model,omitempty"`
	Tokenizer                  string `json:"tokenizer,omitempty"`
	SkipTokenizerInit          bool   `json:"skip_tokenizer_init,omitempty"`
	Revision                   string `json:"revision,omitempty"`
	CodeRevision               string `json:"code_revision,omitempty"`
	TokenizerRevision          string `json:"tokenizer_revision,omitempty"`
	TokenizerMode              string `json:"tokenizer_mode,omitempty"`
	TrustRemoteCode            bool   `json:"trust_remote_code,omitempty"`
	DownloadDir                string `json:"download_dir,omitempty"`
	LoadFormat                 string `json:"load_format,omitempty"`
	ConfigFormat               string `json:"config_format,omitempty"`
	Dtype                      string `json:"dtype,omitempty"`
	KVCacheDtype               string `json:"kv_cache_dtype,omitempty"`
	QuantizationParamPath      string `json:"quantization_param_path,omitempty"`
	Seed                       int    `json:"seed,omitempty"`
	MaxModelLen                int    `json:"max_model_len,omitempty"`
	GuidedDecodingBackend      string `json:"guided_decoding_backend,omitempty"`
	DistributedExecutorBackend string `json:"distributed_executor_backend,omitempty"`
	WorkerUseRay               bool   `json:"worker_use_ray,omitempty"`
	RayWorkersUseNSight        bool   `json:"ray_workers_use_nsight,omitempty"`

	// Performance and serving configuration
	BlockSize                int     `json:"block_size,omitempty"`
	EnablePrefixCaching      bool    `json:"enable_prefix_caching,omitempty"`
	DisableSlidingWindow     bool    `json:"disable_sliding_window,omitempty"`
	UseV2BlockManager        bool    `json:"use_v2_block_manager,omitempty"`
	NumLookaheadSlots        int     `json:"num_lookahead_slots,omitempty"`
	SwapSpace                int     `json:"swap_space,omitempty"`
	CPUOffloadGB             int     `json:"cpu_offload_gb,omitempty"`
	GPUMemoryUtilization     float64 `json:"gpu_memory_utilization,omitempty"`
	NumGPUBlocksOverride     int     `json:"num_gpu_blocks_override,omitempty"`
	MaxNumBatchedTokens      int     `json:"max_num_batched_tokens,omitempty"`
	MaxNumSeqs               int     `json:"max_num_seqs,omitempty"`
	MaxLogprobs              int     `json:"max_logprobs,omitempty"`
	DisableLogStats          bool    `json:"disable_log_stats,omitempty"`
	Quantization             string  `json:"quantization,omitempty"`
	RopeScaling              string  `json:"rope_scaling,omitempty"`
	RopeTheta                float64 `json:"rope_theta,omitempty"`
	EnforceEager             bool    `json:"enforce_eager,omitempty"`
	MaxContextLenToCapture   int     `json:"max_context_len_to_capture,omitempty"`
	MaxSeqLenToCapture       int     `json:"max_seq_len_to_capture,omitempty"`
	DisableCustomAllReduce   bool    `json:"disable_custom_all_reduce,omitempty"`
	TokenizerPoolSize        int     `json:"tokenizer_pool_size,omitempty"`
	TokenizerPoolType        string  `json:"tokenizer_pool_type,omitempty"`
	TokenizerPoolExtraConfig string  `json:"tokenizer_pool_extra_config,omitempty"`
	EnableLoraBias           bool    `json:"enable_lora_bias,omitempty"`
	LoraExtraVocabSize       int     `json:"lora_extra_vocab_size,omitempty"`
	LoraRank                 int     `json:"lora_rank,omitempty"`
	PromptLookbackDistance   int     `json:"prompt_lookback_distance,omitempty"`
	PreemptionMode           string  `json:"preemption_mode,omitempty"`

	// Distributed and parallel processing
	TensorParallelSize            int     `json:"tensor_parallel_size,omitempty"`
	PipelineParallelSize          int     `json:"pipeline_parallel_size,omitempty"`
	MaxParallelLoadingWorkers     int     `json:"max_parallel_loading_workers,omitempty"`
	DisableAsyncOutputProc        bool    `json:"disable_async_output_proc,omitempty"`
	WorkerClass                   string  `json:"worker_class,omitempty"`
	EnabledLoraModules            string  `json:"enabled_lora_modules,omitempty"`
	MaxLoraRank                   int     `json:"max_lora_rank,omitempty"`
	FullyShardedLoras             bool    `json:"fully_sharded_loras,omitempty"`
	LoraModules                   string  `json:"lora_modules,omitempty"`
	PromptAdapters                string  `json:"prompt_adapters,omitempty"`
	MaxPromptAdapterToken         int     `json:"max_prompt_adapter_token,omitempty"`
	Device                        string  `json:"device,omitempty"`
	SchedulerDelay                float64 `json:"scheduler_delay,omitempty"`
	EnableChunkedPrefill          bool    `json:"enable_chunked_prefill,omitempty"`
	SpeculativeModel              string  `json:"speculative_model,omitempty"`
	SpeculativeModelQuantization  string  `json:"speculative_model_quantization,omitempty"`
	SpeculativeRevision           string  `json:"speculative_revision,omitempty"`
	SpeculativeMaxModelLen        int     `json:"speculative_max_model_len,omitempty"`
	SpeculativeDisableByBatchSize int     `json:"speculative_disable_by_batch_size,omitempty"`
	NgptSpeculativeLength         int     `json:"ngpt_speculative_length,omitempty"`
	SpeculativeDisableMqa         bool    `json:"speculative_disable_mqa,omitempty"`
	ModelLoaderExtraConfig        string  `json:"model_loader_extra_config,omitempty"`
	IgnorePatterns                string  `json:"ignore_patterns,omitempty"`
	PreloadedLoraModules          string  `json:"preloaded_lora_modules,omitempty"`

	// OpenAI server specific options
	UDS                            string   `json:"uds,omitempty"`
	UvicornLogLevel                string   `json:"uvicorn_log_level,omitempty"`
	ResponseRole                   string   `json:"response_role,omitempty"`
	SSLKeyfile                     string   `json:"ssl_keyfile,omitempty"`
	SSLCertfile                    string   `json:"ssl_certfile,omitempty"`
	SSLCACerts                     string   `json:"ssl_ca_certs,omitempty"`
	SSLCertReqs                    int      `json:"ssl_cert_reqs,omitempty"`
	RootPath                       string   `json:"root_path,omitempty"`
	Middleware                     []string `json:"middleware,omitempty"`
	ReturnTokensAsTokenIDS         bool     `json:"return_tokens_as_token_ids,omitempty"`
	DisableFrontendMultiprocessing bool     `json:"disable_frontend_multiprocessing,omitempty"`
	EnableAutoToolChoice           bool     `json:"enable_auto_tool_choice,omitempty"`
	ToolCallParser                 string   `json:"tool_call_parser,omitempty"`
	ToolServer                     string   `json:"tool_server,omitempty"`
	ChatTemplate                   string   `json:"chat_template,omitempty"`
	ChatTemplateContentFormat      string   `json:"chat_template_content_format,omitempty"`
	AllowCredentials               bool     `json:"allow_credentials,omitempty"`
	AllowedOrigins                 []string `json:"allowed_origins,omitempty"`
	AllowedMethods                 []string `json:"allowed_methods,omitempty"`
	AllowedHeaders                 []string `json:"allowed_headers,omitempty"`
	APIKey                         []string `json:"api_key,omitempty"`
	EnableLogOutputs               bool     `json:"enable_log_outputs,omitempty"`
	EnableTokenUsage               bool     `json:"enable_token_usage,omitempty"`
	EnableAsyncEngineDebug         bool     `json:"enable_async_engine_debug,omitempty"`
	EngineUseRay                   bool     `json:"engine_use_ray,omitempty"`
	DisableLogRequests             bool     `json:"disable_log_requests,omitempty"`
	MaxLogLen                      int      `json:"max_log_len,omitempty"`

	// Additional engine configuration
	Task                      string `json:"task,omitempty"`
	MultiModalConfig          string `json:"multi_modal_config,omitempty"`
	LimitMmPerPrompt          string `json:"limit_mm_per_prompt,omitempty"`
	EnableSleepMode           bool   `json:"enable_sleep_mode,omitempty"`
	EnableChunkingRequest     bool   `json:"enable_chunking_request,omitempty"`
	CompilationConfig         string `json:"compilation_config,omitempty"`
	DisableSlidingWindowMask  bool   `json:"disable_sliding_window_mask,omitempty"`
	EnableTRTLLMEngineLatency bool   `json:"enable_trtllm_engine_latency,omitempty"`
	OverridePoolingConfig     string `json:"override_pooling_config,omitempty"`
	OverrideNeuronConfig      string `json:"override_neuron_config,omitempty"`
	OverrideKVCacheALIGNSize  int    `json:"override_kv_cache_align_size,omitempty"`
}

// BuildCommandArgs converts VllmServerOptions to command line arguments
// For vLLM native, model is a positional argument after "serve"
func (o *VllmServerOptions) BuildCommandArgs() []string {
	var args []string

	// Add model as positional argument if specified (for native execution)
	if o.Model != "" {
		args = append(args, o.Model)
	}

	// Create a copy without Model field to avoid --model flag
	optionsCopy := *o
	optionsCopy.Model = ""

	// Use package-level multipleFlags variable

	flagArgs := BuildCommandArgs(&optionsCopy, vllmMultiValuedFlags)
	args = append(args, flagArgs...)

	return args
}

func (o *VllmServerOptions) BuildDockerArgs() []string {
	var args []string

	// Use package-level multipleFlags variable
	flagArgs := BuildCommandArgs(o, vllmMultiValuedFlags)
	args = append(args, flagArgs...)

	return args
}

// ParseVllmCommand parses a vLLM serve command string into VllmServerOptions
// Supports multiple formats:
// 1. Full command: "vllm serve --model MODEL_NAME --other-args"
// 2. Full path: "/usr/local/bin/vllm serve --model MODEL_NAME"
// 3. Serve only: "serve --model MODEL_NAME --other-args"
// 4. Args only: "--model MODEL_NAME --other-args"
// 5. Multiline commands with backslashes
func ParseVllmCommand(command string) (*VllmServerOptions, error) {
	executableNames := []string{"vllm"}
	subcommandNames := []string{"serve"}
	multiValuedFlags := map[string]bool{
		"middleware":      true,
		"api_key":         true,
		"allowed_origins": true,
		"allowed_methods": true,
		"allowed_headers": true,
		"lora_modules":    true,
		"prompt_adapters": true,
	}

	var vllmOptions VllmServerOptions
	if err := ParseCommand(command, executableNames, subcommandNames, multiValuedFlags, &vllmOptions); err != nil {
		return nil, err
	}

	return &vllmOptions, nil
}
