package vllm

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

type VllmServerOptions struct {
	// Basic connection options (auto-assigned by llamactl)
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`

	// Model and engine configuration
	Model                      string   `json:"model,omitempty"`
	Tokenizer                  string   `json:"tokenizer,omitempty"`
	SkipTokenizerInit          bool     `json:"skip_tokenizer_init,omitempty"`
	Revision                   string   `json:"revision,omitempty"`
	CodeRevision               string   `json:"code_revision,omitempty"`
	TokenizerRevision          string   `json:"tokenizer_revision,omitempty"`
	TokenizerMode              string   `json:"tokenizer_mode,omitempty"`
	TrustRemoteCode            bool     `json:"trust_remote_code,omitempty"`
	DownloadDir                string   `json:"download_dir,omitempty"`
	LoadFormat                 string   `json:"load_format,omitempty"`
	ConfigFormat               string   `json:"config_format,omitempty"`
	Dtype                      string   `json:"dtype,omitempty"`
	KVCacheDtype               string   `json:"kv_cache_dtype,omitempty"`
	QuantizationParamPath      string   `json:"quantization_param_path,omitempty"`
	Seed                       int      `json:"seed,omitempty"`
	MaxModelLen                int      `json:"max_model_len,omitempty"`
	GuidedDecodingBackend      string   `json:"guided_decoding_backend,omitempty"`
	DistributedExecutorBackend string   `json:"distributed_executor_backend,omitempty"`
	WorkerUseRay               bool     `json:"worker_use_ray,omitempty"`
	RayWorkersUseNSight        bool     `json:"ray_workers_use_nsight,omitempty"`

	// Performance and serving configuration
	BlockSize                    int     `json:"block_size,omitempty"`
	EnablePrefixCaching          bool    `json:"enable_prefix_caching,omitempty"`
	DisableSlidingWindow         bool    `json:"disable_sliding_window,omitempty"`
	UseV2BlockManager            bool    `json:"use_v2_block_manager,omitempty"`
	NumLookaheadSlots            int     `json:"num_lookahead_slots,omitempty"`
	SwapSpace                    int     `json:"swap_space,omitempty"`
	CPUOffloadGB                 int     `json:"cpu_offload_gb,omitempty"`
	GPUMemoryUtilization         float64 `json:"gpu_memory_utilization,omitempty"`
	NumGPUBlocksOverride         int     `json:"num_gpu_blocks_override,omitempty"`
	MaxNumBatchedTokens          int     `json:"max_num_batched_tokens,omitempty"`
	MaxNumSeqs                   int     `json:"max_num_seqs,omitempty"`
	MaxLogprobs                  int     `json:"max_logprobs,omitempty"`
	DisableLogStats              bool    `json:"disable_log_stats,omitempty"`
	Quantization                 string  `json:"quantization,omitempty"`
	RopeScaling                  string  `json:"rope_scaling,omitempty"`
	RopeTheta                    float64 `json:"rope_theta,omitempty"`
	EnforceEager                 bool    `json:"enforce_eager,omitempty"`
	MaxContextLenToCapture       int     `json:"max_context_len_to_capture,omitempty"`
	MaxSeqLenToCapture           int     `json:"max_seq_len_to_capture,omitempty"`
	DisableCustomAllReduce       bool    `json:"disable_custom_all_reduce,omitempty"`
	TokenizerPoolSize            int     `json:"tokenizer_pool_size,omitempty"`
	TokenizerPoolType            string  `json:"tokenizer_pool_type,omitempty"`
	TokenizerPoolExtraConfig     string  `json:"tokenizer_pool_extra_config,omitempty"`
	EnableLoraBias               bool    `json:"enable_lora_bias,omitempty"`
	LoraExtraVocabSize           int     `json:"lora_extra_vocab_size,omitempty"`
	LoraRank                     int     `json:"lora_rank,omitempty"`
	PromptLookbackDistance       int     `json:"prompt_lookback_distance,omitempty"`
	PreemptionMode               string  `json:"preemption_mode,omitempty"`

	// Distributed and parallel processing
	TensorParallelSize             int    `json:"tensor_parallel_size,omitempty"`
	PipelineParallelSize           int    `json:"pipeline_parallel_size,omitempty"`
	MaxParallelLoadingWorkers      int    `json:"max_parallel_loading_workers,omitempty"`
	DisableAsyncOutputProc         bool   `json:"disable_async_output_proc,omitempty"`
	WorkerClass                    string `json:"worker_class,omitempty"`
	EnabledLoraModules             string `json:"enabled_lora_modules,omitempty"`
	MaxLoraRank                    int    `json:"max_lora_rank,omitempty"`
	FullyShardedLoras              bool   `json:"fully_sharded_loras,omitempty"`
	LoraModules                    string `json:"lora_modules,omitempty"`
	PromptAdapters                 string `json:"prompt_adapters,omitempty"`
	MaxPromptAdapterToken          int    `json:"max_prompt_adapter_token,omitempty"`
	Device                         string `json:"device,omitempty"`
	SchedulerDelay                 float64 `json:"scheduler_delay,omitempty"`
	EnableChunkedPrefill           bool   `json:"enable_chunked_prefill,omitempty"`
	SpeculativeModel               string `json:"speculative_model,omitempty"`
	SpeculativeModelQuantization   string `json:"speculative_model_quantization,omitempty"`
	SpeculativeRevision            string `json:"speculative_revision,omitempty"`
	SpeculativeMaxModelLen         int    `json:"speculative_max_model_len,omitempty"`
	SpeculativeDisableByBatchSize  int    `json:"speculative_disable_by_batch_size,omitempty"`
	NgptSpeculativeLength          int    `json:"ngpt_speculative_length,omitempty"`
	SpeculativeDisableMqa          bool   `json:"speculative_disable_mqa,omitempty"`
	ModelLoaderExtraConfig         string `json:"model_loader_extra_config,omitempty"`
	IgnorePatterns                 string `json:"ignore_patterns,omitempty"`
	PreloadedLoraModules           string `json:"preloaded_lora_modules,omitempty"`

	// OpenAI server specific options
	UDS                           string   `json:"uds,omitempty"`
	UvicornLogLevel               string   `json:"uvicorn_log_level,omitempty"`
	ResponseRole                  string   `json:"response_role,omitempty"`
	SSLKeyfile                    string   `json:"ssl_keyfile,omitempty"`
	SSLCertfile                   string   `json:"ssl_certfile,omitempty"`
	SSLCACerts                    string   `json:"ssl_ca_certs,omitempty"`
	SSLCertReqs                   int      `json:"ssl_cert_reqs,omitempty"`
	RootPath                      string   `json:"root_path,omitempty"`
	Middleware                    []string `json:"middleware,omitempty"`
	ReturnTokensAsTokenIDS        bool     `json:"return_tokens_as_token_ids,omitempty"`
	DisableFrontendMultiprocessing bool    `json:"disable_frontend_multiprocessing,omitempty"`
	EnableAutoToolChoice          bool     `json:"enable_auto_tool_choice,omitempty"`
	ToolCallParser                string   `json:"tool_call_parser,omitempty"`
	ToolServer                    string   `json:"tool_server,omitempty"`
	ChatTemplate                  string   `json:"chat_template,omitempty"`
	ChatTemplateContentFormat     string   `json:"chat_template_content_format,omitempty"`
	AllowCredentials              bool     `json:"allow_credentials,omitempty"`
	AllowedOrigins                []string `json:"allowed_origins,omitempty"`
	AllowedMethods                []string `json:"allowed_methods,omitempty"`
	AllowedHeaders                []string `json:"allowed_headers,omitempty"`
	APIKey                        []string `json:"api_key,omitempty"`
	EnableLogOutputs              bool     `json:"enable_log_outputs,omitempty"`
	EnableTokenUsage              bool     `json:"enable_token_usage,omitempty"`
	EnableAsyncEngineDebug        bool     `json:"enable_async_engine_debug,omitempty"`
	EngineUseRay                  bool     `json:"engine_use_ray,omitempty"`
	DisableLogRequests            bool     `json:"disable_log_requests,omitempty"`
	MaxLogLen                     int      `json:"max_log_len,omitempty"`

	// Additional engine configuration
	Task                         string  `json:"task,omitempty"`
	MultiModalConfig             string  `json:"multi_modal_config,omitempty"`
	LimitMmPerPrompt             string  `json:"limit_mm_per_prompt,omitempty"`
	EnableSleepMode              bool    `json:"enable_sleep_mode,omitempty"`
	EnableChunkingRequest        bool    `json:"enable_chunking_request,omitempty"`
	CompilationConfig            string  `json:"compilation_config,omitempty"`
	DisableSlidingWindowMask     bool    `json:"disable_sliding_window_mask,omitempty"`
	EnableTRTLLMEngineLatency    bool    `json:"enable_trtllm_engine_latency,omitempty"`
	OverridePoolingConfig        string  `json:"override_pooling_config,omitempty"`
	OverrideNeuronConfig         string  `json:"override_neuron_config,omitempty"`
	OverrideKVCacheALIGNSize     int     `json:"override_kv_cache_align_size,omitempty"`
}

// NewVllmServerOptions creates a new VllmServerOptions with defaults
func NewVllmServerOptions() *VllmServerOptions {
	return &VllmServerOptions{
		Host:                    "127.0.0.1",
		Port:                    8000,
		TensorParallelSize:      1,
		PipelineParallelSize:    1,
		GPUMemoryUtilization:    0.9,
		BlockSize:              16,
		SwapSpace:              4,
		UvicornLogLevel:         "info",
		ResponseRole:            "assistant",
		TokenizerMode:           "auto",
		TrustRemoteCode:         false,
		EnablePrefixCaching:     false,
		EnforceEager:            false,
		DisableLogStats:         false,
		DisableLogRequests:      false,
		MaxLogprobs:             20,
		EnableLogOutputs:        false,
		EnableTokenUsage:        false,
		AllowCredentials:        false,
		AllowedOrigins:          []string{"*"},
		AllowedMethods:          []string{"*"},
		AllowedHeaders:          []string{"*"},
	}
}

// UnmarshalJSON implements custom JSON unmarshaling to support multiple field names
func (o *VllmServerOptions) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to handle multiple field names
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create a temporary struct for standard unmarshaling
	type tempOptions VllmServerOptions
	temp := tempOptions{}

	// Standard unmarshal first
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy to our struct
	*o = VllmServerOptions(temp)

	// Handle alternative field names (CLI format with dashes)
	fieldMappings := map[string]string{
		// Basic options
		"tensor-parallel-size":             "tensor_parallel_size",
		"pipeline-parallel-size":           "pipeline_parallel_size",
		"max-parallel-loading-workers":     "max_parallel_loading_workers",
		"disable-async-output-proc":        "disable_async_output_proc",
		"worker-class":                     "worker_class",
		"enabled-lora-modules":             "enabled_lora_modules",
		"max-lora-rank":                    "max_lora_rank",
		"fully-sharded-loras":              "fully_sharded_loras",
		"lora-modules":                     "lora_modules",
		"prompt-adapters":                  "prompt_adapters",
		"max-prompt-adapter-token":         "max_prompt_adapter_token",
		"scheduler-delay":                  "scheduler_delay",
		"enable-chunked-prefill":           "enable_chunked_prefill",
		"speculative-model":                "speculative_model",
		"speculative-model-quantization":   "speculative_model_quantization",
		"speculative-revision":             "speculative_revision",
		"speculative-max-model-len":        "speculative_max_model_len",
		"speculative-disable-by-batch-size": "speculative_disable_by_batch_size",
		"ngpt-speculative-length":          "ngpt_speculative_length",
		"speculative-disable-mqa":          "speculative_disable_mqa",
		"model-loader-extra-config":        "model_loader_extra_config",
		"ignore-patterns":                  "ignore_patterns",
		"preloaded-lora-modules":           "preloaded_lora_modules",

		// Model configuration
		"skip-tokenizer-init":              "skip_tokenizer_init",
		"code-revision":                    "code_revision",
		"tokenizer-revision":               "tokenizer_revision",
		"tokenizer-mode":                   "tokenizer_mode",
		"trust-remote-code":                "trust_remote_code",
		"download-dir":                     "download_dir",
		"load-format":                      "load_format",
		"config-format":                    "config_format",
		"kv-cache-dtype":                   "kv_cache_dtype",
		"quantization-param-path":          "quantization_param_path",
		"max-model-len":                    "max_model_len",
		"guided-decoding-backend":          "guided_decoding_backend",
		"distributed-executor-backend":     "distributed_executor_backend",
		"worker-use-ray":                   "worker_use_ray",
		"ray-workers-use-nsight":           "ray_workers_use_nsight",

		// Performance configuration
		"block-size":                       "block_size",
		"enable-prefix-caching":            "enable_prefix_caching",
		"disable-sliding-window":           "disable_sliding_window",
		"use-v2-block-manager":             "use_v2_block_manager",
		"num-lookahead-slots":              "num_lookahead_slots",
		"swap-space":                       "swap_space",
		"cpu-offload-gb":                   "cpu_offload_gb",
		"gpu-memory-utilization":           "gpu_memory_utilization",
		"num-gpu-blocks-override":          "num_gpu_blocks_override",
		"max-num-batched-tokens":           "max_num_batched_tokens",
		"max-num-seqs":                     "max_num_seqs",
		"max-logprobs":                     "max_logprobs",
		"disable-log-stats":                "disable_log_stats",
		"rope-scaling":                     "rope_scaling",
		"rope-theta":                       "rope_theta",
		"enforce-eager":                    "enforce_eager",
		"max-context-len-to-capture":       "max_context_len_to_capture",
		"max-seq-len-to-capture":           "max_seq_len_to_capture",
		"disable-custom-all-reduce":        "disable_custom_all_reduce",
		"tokenizer-pool-size":              "tokenizer_pool_size",
		"tokenizer-pool-type":              "tokenizer_pool_type",
		"tokenizer-pool-extra-config":      "tokenizer_pool_extra_config",
		"enable-lora-bias":                 "enable_lora_bias",
		"lora-extra-vocab-size":            "lora_extra_vocab_size",
		"lora-rank":                        "lora_rank",
		"prompt-lookback-distance":         "prompt_lookback_distance",
		"preemption-mode":                  "preemption_mode",

		// Server configuration
		"uvicorn-log-level":                  "uvicorn_log_level",
		"response-role":                      "response_role",
		"ssl-keyfile":                        "ssl_keyfile",
		"ssl-certfile":                       "ssl_certfile",
		"ssl-ca-certs":                       "ssl_ca_certs",
		"ssl-cert-reqs":                      "ssl_cert_reqs",
		"root-path":                          "root_path",
		"return-tokens-as-token-ids":         "return_tokens_as_token_ids",
		"disable-frontend-multiprocessing":   "disable_frontend_multiprocessing",
		"enable-auto-tool-choice":            "enable_auto_tool_choice",
		"tool-call-parser":                   "tool_call_parser",
		"tool-server":                        "tool_server",
		"chat-template":                      "chat_template",
		"chat-template-content-format":       "chat_template_content_format",
		"allow-credentials":                  "allow_credentials",
		"allowed-origins":                    "allowed_origins",
		"allowed-methods":                    "allowed_methods",
		"allowed-headers":                    "allowed_headers",
		"api-key":                            "api_key",
		"enable-log-outputs":                 "enable_log_outputs",
		"enable-token-usage":                 "enable_token_usage",
		"enable-async-engine-debug":          "enable_async_engine_debug",
		"engine-use-ray":                     "engine_use_ray",
		"disable-log-requests":               "disable_log_requests",
		"max-log-len":                        "max_log_len",

		// Additional options
		"multi-modal-config":               "multi_modal_config",
		"limit-mm-per-prompt":              "limit_mm_per_prompt",
		"enable-sleep-mode":                "enable_sleep_mode",
		"enable-chunking-request":          "enable_chunking_request",
		"compilation-config":               "compilation_config",
		"disable-sliding-window-mask":      "disable_sliding_window_mask",
		"enable-trtllm-engine-latency":     "enable_trtllm_engine_latency",
		"override-pooling-config":          "override_pooling_config",
		"override-neuron-config":           "override_neuron_config",
		"override-kv-cache-align-size":     "override_kv_cache_align_size",
	}

	// Process alternative field names
	for altName, canonicalName := range fieldMappings {
		if value, exists := raw[altName]; exists {
			// Use reflection to set the field value
			v := reflect.ValueOf(o).Elem()
			field := v.FieldByNameFunc(func(fieldName string) bool {
				field, _ := v.Type().FieldByName(fieldName)
				jsonTag := field.Tag.Get("json")
				return jsonTag == canonicalName+",omitempty" || jsonTag == canonicalName
			})

			if field.IsValid() && field.CanSet() {
				switch field.Kind() {
				case reflect.Int:
					if intVal, ok := value.(float64); ok {
						field.SetInt(int64(intVal))
					} else if strVal, ok := value.(string); ok {
						if intVal, err := strconv.Atoi(strVal); err == nil {
							field.SetInt(int64(intVal))
						}
					}
				case reflect.Float64:
					if floatVal, ok := value.(float64); ok {
						field.SetFloat(floatVal)
					} else if strVal, ok := value.(string); ok {
						if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
							field.SetFloat(floatVal)
						}
					}
				case reflect.String:
					if strVal, ok := value.(string); ok {
						field.SetString(strVal)
					}
				case reflect.Bool:
					if boolVal, ok := value.(bool); ok {
						field.SetBool(boolVal)
					}
				case reflect.Slice:
					if field.Type().Elem().Kind() == reflect.String {
						if strVal, ok := value.(string); ok {
							// Split comma-separated values
							values := strings.Split(strVal, ",")
							for i, v := range values {
								values[i] = strings.TrimSpace(v)
							}
							field.Set(reflect.ValueOf(values))
						} else if slice, ok := value.([]interface{}); ok {
							var strSlice []string
							for _, item := range slice {
								if str, ok := item.(string); ok {
									strSlice = append(strSlice, str)
								}
							}
							field.Set(reflect.ValueOf(strSlice))
						}
					}
				}
			}
		}
	}

	return nil
}

// BuildCommandArgs converts VllmServerOptions to command line arguments
// Note: This does NOT include the "serve" subcommand, that's handled at the instance level
func (o *VllmServerOptions) BuildCommandArgs() []string {
	var args []string

	v := reflect.ValueOf(o).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get the JSON tag to determine the flag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Remove ",omitempty" from the tag
		flagName := jsonTag
		if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
			flagName = jsonTag[:commaIndex]
		}

		// Skip host and port as they are handled by llamactl
		if flagName == "host" || flagName == "port" {
			continue
		}

		// Convert snake_case to kebab-case for CLI flags
		flagName = strings.ReplaceAll(flagName, "_", "-")

		// Add the appropriate arguments based on field type and value
		switch field.Kind() {
		case reflect.Bool:
			if field.Bool() {
				args = append(args, "--"+flagName)
			}
		case reflect.Int:
			if field.Int() != 0 {
				args = append(args, "--"+flagName, strconv.FormatInt(field.Int(), 10))
			}
		case reflect.Float64:
			if field.Float() != 0 {
				args = append(args, "--"+flagName, strconv.FormatFloat(field.Float(), 'f', -1, 64))
			}
		case reflect.String:
			if field.String() != "" {
				args = append(args, "--"+flagName, field.String())
			}
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				// Handle []string fields - some are comma-separated, some use multiple flags
				if flagName == "api-key" || flagName == "allowed-origins" || flagName == "allowed-methods" || flagName == "allowed-headers" || flagName == "middleware" {
					// Multiple flags for these
					for j := 0; j < field.Len(); j++ {
						args = append(args, "--"+flagName, field.Index(j).String())
					}
				} else {
					// Comma-separated for others
					if field.Len() > 0 {
						var values []string
						for j := 0; j < field.Len(); j++ {
							values = append(values, field.Index(j).String())
						}
						args = append(args, "--"+flagName, strings.Join(values, ","))
					}
				}
			}
		}
	}

	return args
}