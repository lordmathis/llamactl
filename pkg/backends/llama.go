package backends

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/validation"
	"reflect"
	"strconv"
)

// llamaMultiValuedFlags defines flags that should be repeated for each value rather than comma-separated
// Used for both parsing (with underscores) and building (with dashes)
var llamaMultiValuedFlags = map[string]bool{
	// Parsing keys (with underscores)
	"override_tensor":       true,
	"override_kv":           true,
	"lora":                  true,
	"lora_scaled":           true,
	"control_vector":        true,
	"control_vector_scaled": true,
	"dry_sequence_breaker":  true,
	"logit_bias":            true,
	// Building keys (with dashes)
	"override-tensor":       true,
	"override-kv":           true,
	"lora-scaled":           true,
	"control-vector":        true,
	"control-vector-scaled": true,
	"dry-sequence-breaker":  true,
	"logit-bias":            true,
}

type LlamaServerOptions struct {
	// Common params
	VerbosePrompt           bool     `json:"verbose_prompt,omitempty"`
	Threads                 int      `json:"threads,omitempty"`
	ThreadsBatch            int      `json:"threads_batch,omitempty"`
	CPUMask                 string   `json:"cpu_mask,omitempty"`
	CPURange                string   `json:"cpu_range,omitempty"`
	CPUStrict               int      `json:"cpu_strict,omitempty"`
	Prio                    int      `json:"prio,omitempty"`
	Poll                    int      `json:"poll,omitempty"`
	CPUMaskBatch            string   `json:"cpu_mask_batch,omitempty"`
	CPURangeBatch           string   `json:"cpu_range_batch,omitempty"`
	CPUStrictBatch          int      `json:"cpu_strict_batch,omitempty"`
	PrioBatch               int      `json:"prio_batch,omitempty"`
	PollBatch               int      `json:"poll_batch,omitempty"`
	CtxSize                 int      `json:"ctx_size,omitempty"`
	Predict                 int      `json:"predict,omitempty"`
	BatchSize               int      `json:"batch_size,omitempty"`
	UBatchSize              int      `json:"ubatch_size,omitempty"`
	Keep                    int      `json:"keep,omitempty"`
	FlashAttn               bool     `json:"flash_attn,omitempty"`
	NoPerf                  bool     `json:"no_perf,omitempty"`
	Escape                  bool     `json:"escape,omitempty"`
	NoEscape                bool     `json:"no_escape,omitempty"`
	RopeScaling             string   `json:"rope_scaling,omitempty"`
	RopeScale               float64  `json:"rope_scale,omitempty"`
	RopeFreqBase            float64  `json:"rope_freq_base,omitempty"`
	RopeFreqScale           float64  `json:"rope_freq_scale,omitempty"`
	YarnOrigCtx             int      `json:"yarn_orig_ctx,omitempty"`
	YarnExtFactor           float64  `json:"yarn_ext_factor,omitempty"`
	YarnAttnFactor          float64  `json:"yarn_attn_factor,omitempty"`
	YarnBetaSlow            float64  `json:"yarn_beta_slow,omitempty"`
	YarnBetaFast            float64  `json:"yarn_beta_fast,omitempty"`
	DumpKVCache             bool     `json:"dump_kv_cache,omitempty"`
	NoKVOffload             bool     `json:"no_kv_offload,omitempty"`
	CacheTypeK              string   `json:"cache_type_k,omitempty"`
	CacheTypeV              string   `json:"cache_type_v,omitempty"`
	DefragThold             float64  `json:"defrag_thold,omitempty"`
	Parallel                int      `json:"parallel,omitempty"`
	Mlock                   bool     `json:"mlock,omitempty"`
	NoMmap                  bool     `json:"no_mmap,omitempty"`
	Numa                    string   `json:"numa,omitempty"`
	Device                  string   `json:"device,omitempty"`
	OverrideTensor          []string `json:"override_tensor,omitempty"`
	GPULayers               int      `json:"gpu_layers,omitempty"`
	SplitMode               string   `json:"split_mode,omitempty"`
	TensorSplit             string   `json:"tensor_split,omitempty"`
	MainGPU                 int      `json:"main_gpu,omitempty"`
	CheckTensors            bool     `json:"check_tensors,omitempty"`
	OverrideKV              []string `json:"override_kv,omitempty"`
	Lora                    []string `json:"lora,omitempty"`
	LoraScaled              []string `json:"lora_scaled,omitempty"`
	ControlVector           []string `json:"control_vector,omitempty"`
	ControlVectorScaled     []string `json:"control_vector_scaled,omitempty"`
	ControlVectorLayerRange string   `json:"control_vector_layer_range,omitempty"`
	Model                   string   `json:"model,omitempty"`
	ModelURL                string   `json:"model_url,omitempty"`
	HFRepo                  string   `json:"hf_repo,omitempty"`
	HFRepoDraft             string   `json:"hf_repo_draft,omitempty"`
	HFFile                  string   `json:"hf_file,omitempty"`
	HFRepoV                 string   `json:"hf_repo_v,omitempty"`
	HFFileV                 string   `json:"hf_file_v,omitempty"`
	HFToken                 string   `json:"hf_token,omitempty"`
	LogDisable              bool     `json:"log_disable,omitempty"`
	LogFile                 string   `json:"log_file,omitempty"`
	LogColors               bool     `json:"log_colors,omitempty"`
	Verbose                 bool     `json:"verbose,omitempty"`
	Verbosity               int      `json:"verbosity,omitempty"`
	LogPrefix               bool     `json:"log_prefix,omitempty"`
	LogTimestamps           bool     `json:"log_timestamps,omitempty"`

	// Sampling params
	Samplers           string   `json:"samplers,omitempty"`
	Seed               int      `json:"seed,omitempty"`
	SamplingSeq        string   `json:"sampling_seq,omitempty"`
	IgnoreEOS          bool     `json:"ignore_eos,omitempty"`
	Temperature        float64  `json:"temp,omitempty"`
	TopK               int      `json:"top_k,omitempty"`
	TopP               float64  `json:"top_p,omitempty"`
	MinP               float64  `json:"min_p,omitempty"`
	XTCProbability     float64  `json:"xtc_probability,omitempty"`
	XTCThreshold       float64  `json:"xtc_threshold,omitempty"`
	Typical            float64  `json:"typical,omitempty"`
	RepeatLastN        int      `json:"repeat_last_n,omitempty"`
	RepeatPenalty      float64  `json:"repeat_penalty,omitempty"`
	PresencePenalty    float64  `json:"presence_penalty,omitempty"`
	FrequencyPenalty   float64  `json:"frequency_penalty,omitempty"`
	DryMultiplier      float64  `json:"dry_multiplier,omitempty"`
	DryBase            float64  `json:"dry_base,omitempty"`
	DryAllowedLength   int      `json:"dry_allowed_length,omitempty"`
	DryPenaltyLastN    int      `json:"dry_penalty_last_n,omitempty"`
	DrySequenceBreaker []string `json:"dry_sequence_breaker,omitempty"`
	DynatempRange      float64  `json:"dynatemp_range,omitempty"`
	DynatempExp        float64  `json:"dynatemp_exp,omitempty"`
	Mirostat           int      `json:"mirostat,omitempty"`
	MirostatLR         float64  `json:"mirostat_lr,omitempty"`
	MirostatEnt        float64  `json:"mirostat_ent,omitempty"`
	LogitBias          []string `json:"logit_bias,omitempty"`
	Grammar            string   `json:"grammar,omitempty"`
	GrammarFile        string   `json:"grammar_file,omitempty"`
	JSONSchema         string   `json:"json_schema,omitempty"`
	JSONSchemaFile     string   `json:"json_schema_file,omitempty"`

	// Example-specific params
	NoContextShift       bool    `json:"no_context_shift,omitempty"`
	Special              bool    `json:"special,omitempty"`
	NoWarmup             bool    `json:"no_warmup,omitempty"`
	SPMInfill            bool    `json:"spm_infill,omitempty"`
	Pooling              string  `json:"pooling,omitempty"`
	ContBatching         bool    `json:"cont_batching,omitempty"`
	NoContBatching       bool    `json:"no_cont_batching,omitempty"`
	MMProj               string  `json:"mmproj,omitempty"`
	MMProjURL            string  `json:"mmproj_url,omitempty"`
	NoMMProj             bool    `json:"no_mmproj,omitempty"`
	NoMMProjOffload      bool    `json:"no_mmproj_offload,omitempty"`
	Alias                string  `json:"alias,omitempty"`
	Host                 string  `json:"host,omitempty"`
	Port                 int     `json:"port,omitempty"`
	Path                 string  `json:"path,omitempty"`
	NoWebUI              bool    `json:"no_webui,omitempty"`
	Embedding            bool    `json:"embedding,omitempty"`
	Reranking            bool    `json:"reranking,omitempty"`
	APIKey               string  `json:"api_key,omitempty"`
	APIKeyFile           string  `json:"api_key_file,omitempty"`
	SSLKeyFile           string  `json:"ssl_key_file,omitempty"`
	SSLCertFile          string  `json:"ssl_cert_file,omitempty"`
	ChatTemplateKwargs   string  `json:"chat_template_kwargs,omitempty"`
	Timeout              int     `json:"timeout,omitempty"`
	ThreadsHTTP          int     `json:"threads_http,omitempty"`
	CacheReuse           int     `json:"cache_reuse,omitempty"`
	Metrics              bool    `json:"metrics,omitempty"`
	Slots                bool    `json:"slots,omitempty"`
	Props                bool    `json:"props,omitempty"`
	NoSlots              bool    `json:"no_slots,omitempty"`
	SlotSavePath         string  `json:"slot_save_path,omitempty"`
	Jinja                bool    `json:"jinja,omitempty"`
	ReasoningFormat      string  `json:"reasoning_format,omitempty"`
	ReasoningBudget      int     `json:"reasoning_budget,omitempty"`
	ChatTemplate         string  `json:"chat_template,omitempty"`
	ChatTemplateFile     string  `json:"chat_template_file,omitempty"`
	NoPrefillAssistant   bool    `json:"no_prefill_assistant,omitempty"`
	SlotPromptSimilarity float64 `json:"slot_prompt_similarity,omitempty"`
	LoraInitWithoutApply bool    `json:"lora_init_without_apply,omitempty"`
	DraftMax             int     `json:"draft_max,omitempty"`
	DraftMin             int     `json:"draft_min,omitempty"`
	DraftPMin            float64 `json:"draft_p_min,omitempty"`
	CtxSizeDraft         int     `json:"ctx_size_draft,omitempty"`
	DeviceDraft          string  `json:"device_draft,omitempty"`
	GPULayersDraft       int     `json:"gpu_layers_draft,omitempty"`
	ModelDraft           string  `json:"model_draft,omitempty"`
	CacheTypeKDraft      string  `json:"cache_type_k_draft,omitempty"`
	CacheTypeVDraft      string  `json:"cache_type_v_draft,omitempty"`

	// Audio/TTS params
	ModelVocoder      string `json:"model_vocoder,omitempty"`
	TTSUseGuideTokens bool   `json:"tts_use_guide_tokens,omitempty"`

	// Default model params
	EmbdBGESmallEnDefault bool `json:"embd_bge_small_en_default,omitempty"`
	EmbdE5SmallEnDefault  bool `json:"embd_e5_small_en_default,omitempty"`
	EmbdGTESmallDefault   bool `json:"embd_gte_small_default,omitempty"`
	FIMQwen1_5BDefault    bool `json:"fim_qwen_1_5b_default,omitempty"`
	FIMQwen3BDefault      bool `json:"fim_qwen_3b_default,omitempty"`
	FIMQwen7BDefault      bool `json:"fim_qwen_7b_default,omitempty"`
	FIMQwen7BSpec         bool `json:"fim_qwen_7b_spec,omitempty"`
	FIMQwen14BSpec        bool `json:"fim_qwen_14b_spec,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to support multiple field names
func (o *LlamaServerOptions) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to handle multiple field names
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create a temporary struct for standard unmarshaling
	type tempOptions LlamaServerOptions
	temp := tempOptions{}

	// Standard unmarshal first
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy to our struct
	*o = LlamaServerOptions(temp)

	// Handle alternative field names
	fieldMappings := map[string]string{
		// Common params
		"t":             "threads",         // -t, --threads N
		"tb":            "threads_batch",   // -tb, --threads-batch N
		"C":             "cpu_mask",        // -C, --cpu-mask M
		"Cr":            "cpu_range",       // -Cr, --cpu-range lo-hi
		"Cb":            "cpu_mask_batch",  // -Cb, --cpu-mask-batch M
		"Crb":           "cpu_range_batch", // -Crb, --cpu-range-batch lo-hi
		"c":             "ctx_size",        // -c, --ctx-size N
		"n":             "predict",         // -n, --predict N
		"n-predict":     "predict",         // --n-predict N
		"b":             "batch_size",      // -b, --batch-size N
		"ub":            "ubatch_size",     // -ub, --ubatch-size N
		"fa":            "flash_attn",      // -fa, --flash-attn
		"e":             "escape",          // -e, --escape
		"dkvc":          "dump_kv_cache",   // -dkvc, --dump-kv-cache
		"nkvo":          "no_kv_offload",   // -nkvo, --no-kv-offload
		"ctk":           "cache_type_k",    // -ctk, --cache-type-k TYPE
		"ctv":           "cache_type_v",    // -ctv, --cache-type-v TYPE
		"dt":            "defrag_thold",    // -dt, --defrag-thold N
		"np":            "parallel",        // -np, --parallel N
		"dev":           "device",          // -dev, --device <dev1,dev2,..>
		"ot":            "override_tensor", // --override-tensor, -ot
		"ngl":           "gpu_layers",      // -ngl, --gpu-layers, --n-gpu-layers N
		"n-gpu-layers":  "gpu_layers",      // --n-gpu-layers N
		"sm":            "split_mode",      // -sm, --split-mode
		"ts":            "tensor_split",    // -ts, --tensor-split N0,N1,N2,...
		"mg":            "main_gpu",        // -mg, --main-gpu INDEX
		"m":             "model",           // -m, --model FNAME
		"mu":            "model_url",       // -mu, --model-url MODEL_URL
		"hf":            "hf_repo",         // -hf, -hfr, --hf-repo
		"hfr":           "hf_repo",         // -hf, -hfr, --hf-repo
		"hfd":           "hf_repo_draft",   // -hfd, -hfrd, --hf-repo-draft
		"hfrd":          "hf_repo_draft",   // -hfd, -hfrd, --hf-repo-draft
		"hff":           "hf_file",         // -hff, --hf-file FILE
		"hfv":           "hf_repo_v",       // -hfv, -hfrv, --hf-repo-v
		"hfrv":          "hf_repo_v",       // -hfv, -hfrv, --hf-repo-v
		"hffv":          "hf_file_v",       // -hffv, --hf-file-v FILE
		"hft":           "hf_token",        // -hft, --hf-token TOKEN
		"v":             "verbose",         // -v, --verbose, --log-verbose
		"log-verbose":   "verbose",         // --log-verbose
		"lv":            "verbosity",       // -lv, --verbosity, --log-verbosity N
		"log-verbosity": "verbosity",       // --log-verbosity N

		// Sampling params
		"s":  "seed",             // -s, --seed SEED
		"l":  "logit_bias",       // -l, --logit-bias
		"j":  "json_schema",      // -j, --json-schema SCHEMA
		"jf": "json_schema_file", // -jf, --json-schema-file FILE

		// Example-specific params
		"sp":                 "special",                // -sp, --special
		"cb":                 "cont_batching",          // -cb, --cont-batching
		"nocb":               "no_cont_batching",       // -nocb, --no-cont-batching
		"a":                  "alias",                  // -a, --alias STRING
		"embeddings":         "embedding",              // --embeddings
		"rerank":             "reranking",              // --reranking
		"to":                 "timeout",                // -to, --timeout N
		"sps":                "slot_prompt_similarity", // -sps, --slot-prompt-similarity
		"draft":              "draft-max",              // -draft, --draft-max N
		"draft-n":            "draft-max",              // --draft-n-max N
		"draft-n-min":        "draft_min",              // --draft-n-min N
		"cd":                 "ctx_size_draft",         // -cd, --ctx-size-draft N
		"devd":               "device_draft",           // -devd, --device-draft
		"ngld":               "gpu_layers_draft",       // -ngld, --gpu-layers-draft
		"n-gpu-layers-draft": "gpu_layers_draft",       // --n-gpu-layers-draft N
		"md":                 "model_draft",            // -md, --model-draft FNAME
		"ctkd":               "cache_type_k_draft",     // -ctkd, --cache-type-k-draft TYPE
		"ctvd":               "cache_type_v_draft",     // -ctvd, --cache-type-v-draft TYPE
		"mv":                 "model_vocoder",          // -mv, --model-vocoder FNAME
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
				}
			}
		}
	}

	return nil
}

func (o *LlamaServerOptions) GetPort() int {
	return o.Port
}

func (o *LlamaServerOptions) SetPort(port int) {
	o.Port = port
}

func (o *LlamaServerOptions) GetHost() string {
	return o.Host
}

func (o *LlamaServerOptions) Validate() error {
	if o == nil {
		return validation.ValidationError(fmt.Errorf("llama server options cannot be nil for llama.cpp backend"))
	}

	// Use reflection to check all string fields for injection patterns
	if err := validation.ValidateStructStrings(o, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if o.Port < 0 || o.Port > 65535 {
		return validation.ValidationError(fmt.Errorf("invalid port range: %d", o.Port))
	}

	return nil
}

// BuildCommandArgs converts InstanceOptions to command line arguments
func (o *LlamaServerOptions) BuildCommandArgs() []string {
	// Llama uses multiple flags for arrays by default (not comma-separated)
	// Use package-level llamaMultiValuedFlags variable
	return BuildCommandArgs(o, llamaMultiValuedFlags)
}

func (o *LlamaServerOptions) BuildDockerArgs() []string {
	// For llama, Docker args are the same as normal args
	return o.BuildCommandArgs()
}

// ParseLlamaCommand parses a llama-server command string into LlamaServerOptions
// Supports multiple formats:
// 1. Full command: "llama-server --model file.gguf"
// 2. Full path: "/usr/local/bin/llama-server --model file.gguf"
// 3. Args only: "--model file.gguf --gpu-layers 32"
// 4. Multiline commands with backslashes
func ParseLlamaCommand(command string) (*LlamaServerOptions, error) {
	executableNames := []string{"llama-server"}
	var subcommandNames []string // Llama has no subcommands
	// Use package-level llamaMultiValuedFlags variable

	var llamaOptions LlamaServerOptions
	if err := ParseCommand(command, executableNames, subcommandNames, llamaMultiValuedFlags, &llamaOptions); err != nil {
		return nil, err
	}

	return &llamaOptions, nil
}
