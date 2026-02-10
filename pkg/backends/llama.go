package backends

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/validation"
	"reflect"
)

// llamaMultiValuedFlags defines flags that should be repeated for each value rather than comma-separated
// Keys use snake_case as the parser converts kebab-case flags to snake_case before lookup
var llamaMultiValuedFlags = map[string]struct{}{
	"override_tensor":       {},
	"override_tensor_draft": {},
	"override_kv":           {},
	"lora":                  {},
	"lora_scaled":           {},
	"control_vector":        {},
	"control_vector_scaled": {},
	"dry_sequence_breaker":  {},
	"logit_bias":            {},
}

type LlamaServerOptions struct {
	// Common params
	VerbosePrompt    bool    `json:"verbose_prompt,omitempty"`
	Threads          int     `json:"threads,omitempty"`         // -t, --threads N
	ThreadsBatch     int     `json:"threads_batch,omitempty"`   // -tb, --threads-batch N
	CPUMask          string  `json:"cpu_mask,omitempty"`        // -C, --cpu-mask M
	CPURange         string  `json:"cpu_range,omitempty"`       // -Cr, --cpu-range lo-hi
	CPUStrict        int     `json:"cpu_strict,omitempty"`      // --cpu-strict <0|1>
	Prio             int     `json:"prio,omitempty"`            // --prio N
	Poll             int     `json:"poll,omitempty"`            // --poll <0...100>
	CPUMaskBatch     string  `json:"cpu_mask_batch,omitempty"`  // -Cb, --cpu-mask-batch M
	CPURangeBatch    string  `json:"cpu_range_batch,omitempty"` // -Crb, --cpu-range-batch lo-hi
	CPUStrictBatch   int     `json:"cpu_strict_batch,omitempty"`
	PrioBatch        int     `json:"prio_batch,omitempty"`
	PollBatch        int     `json:"poll_batch,omitempty"`
	CtxSize          int     `json:"ctx_size,omitempty"`      // -c, --ctx-size N
	Predict          int     `json:"predict,omitempty"`       // -n, --predict, --n-predict N
	BatchSize        int     `json:"batch_size,omitempty"`    // -b, --batch-size N
	UBatchSize       int     `json:"ubatch_size,omitempty"`   // -ub, --ubatch-size N
	Keep             int     `json:"keep,omitempty"`          // --keep N
	SWAFull          bool    `json:"swa_full,omitempty"`      // --swa-full
	FlashAttn        string  `json:"flash_attn,omitempty"`    // -fa, --flash-attn [on|off|auto]
	Perf             bool    `json:"perf,omitempty"`          // --perf
	NoPerf           bool    `json:"no_perf,omitempty"`       // --no-perf
	Escape           bool    `json:"escape,omitempty"`        // -e, --escape
	NoEscape         bool    `json:"no_escape,omitempty"`     // --no-escape
	RopeScaling      string  `json:"rope_scaling,omitempty"`  // --rope-scaling {none,linear,yarn}
	RopeScale        float64 `json:"rope_scale,omitempty"`    // --rope-scale N
	RopeFreqBase     float64 `json:"rope_freq_base,omitempty"`
	RopeFreqScale    float64 `json:"rope_freq_scale,omitempty"`
	YarnOrigCtx      int     `json:"yarn_orig_ctx,omitempty"`
	YarnExtFactor    float64 `json:"yarn_ext_factor,omitempty"`
	YarnAttnFactor   float64 `json:"yarn_attn_factor,omitempty"`
	YarnBetaSlow     float64 `json:"yarn_beta_slow,omitempty"`
	YarnBetaFast     float64 `json:"yarn_beta_fast,omitempty"`
	KVOffload        bool    `json:"kv_offload,omitempty"`    // -kvo, --kv-offload
	NoKVOffload      bool    `json:"no_kv_offload,omitempty"` // -nkvo, --no-kv-offload
	Repack           bool    `json:"repack,omitempty"`        // --repack
	NoRepack         bool    `json:"no_repack,omitempty"`     // -nr, --no-repack
	NoHost           bool    `json:"no_host,omitempty"`       // --no-host
	CacheTypeK       string  `json:"cache_type_k,omitempty"`  // -ctk, --cache-type-k TYPE
	CacheTypeV       string  `json:"cache_type_v,omitempty"`  // -ctv, --cache-type-v TYPE
	DefragThold      float64 `json:"defrag_thold,omitempty"`  // -dt, --defrag-thold N
	Mlock            bool    `json:"mlock,omitempty"`         // --mlock
	Mmap             bool    `json:"mmap,omitempty"`          // --mmap
	NoMmap           bool    `json:"no_mmap,omitempty"`       // --no-mmap
	DirectIO         bool    `json:"direct_io,omitempty"`     // -dio, --direct-io
	NoDirectIO       bool    `json:"no_direct_io,omitempty"`  // -ndio, --no-direct-io
	Numa             string  `json:"numa,omitempty"`          // --numa TYPE
	Device           string  `json:"device,omitempty"`        // -dev, --device <dev1,dev2,..>
	OverrideTensor   []string `json:"override_tensor,omitempty"` // -ot, --override-tensor
	CPUMoe           bool     `json:"cpu_moe,omitempty"`         // -cmoe, --cpu-moe
	NCPUMoe          int      `json:"n_cpu_moe,omitempty"`       // -ncmoe, --n-cpu-moe N
	GPULayers        int      `json:"gpu_layers,omitempty"`      // -ngl, --gpu-layers, --n-gpu-layers N
	SplitMode        string   `json:"split_mode,omitempty"`      // -sm, --split-mode {none,layer,row}
	TensorSplit      string   `json:"tensor_split,omitempty"`    // -ts, --tensor-split N0,N1,N2,...
	MainGPU          int      `json:"main_gpu,omitempty"`        // -mg, --main-gpu INDEX
	Fit              string   `json:"fit,omitempty"`             // -fit, --fit [on|off]
	FitTarget        string   `json:"fit_target,omitempty"`      // -fitt, --fit-target MiB0,MiB1,MiB2,...
	FitCtx           int      `json:"fit_ctx,omitempty"`         // -fitc, --fit-ctx N
	CheckTensors     bool     `json:"check_tensors,omitempty"`   // --check-tensors
	OverrideKV       []string `json:"override_kv,omitempty"`     // --override-kv KEY=TYPE:VALUE,...
	OpOffload        bool     `json:"op_offload,omitempty"`      // --op-offload
	NoOpOffload      bool     `json:"no_op_offload,omitempty"`   // --no-op-offload
	Lora             []string `json:"lora,omitempty"`            // --lora FNAME
	LoraScaled       []string `json:"lora_scaled,omitempty"`     // --lora-scaled FNAME:SCALE,...
	ControlVector    []string `json:"control_vector,omitempty"`  // --control-vector FNAME
	ControlVectorScaled     []string `json:"control_vector_scaled,omitempty"`      // --control-vector-scaled FNAME:SCALE,...
	ControlVectorLayerRange string   `json:"control_vector_layer_range,omitempty"` // --control-vector-layer-range START END
	Model                   string   `json:"model,omitempty"`                      // -m, --model FNAME
	ModelURL                string   `json:"model_url,omitempty"`                  // -mu, --model-url MODEL_URL
	DockerRepo              string   `json:"docker_repo,omitempty"`                // -dr, --docker-repo [<repo>/]<model>[:quant]
	HFRepo                  string   `json:"hf_repo,omitempty"`                    // -hf, -hfr, --hf-repo <user>/<model>[:quant]
	HFRepoDraft             string   `json:"hf_repo_draft,omitempty"`              // -hfd, -hfrd, --hf-repo-draft <user>/<model>[:quant]
	HFFile                  string   `json:"hf_file,omitempty"`                    // -hff, --hf-file FILE
	HFRepoV                 string   `json:"hf_repo_v,omitempty"`                  // -hfv, -hfrv, --hf-repo-v <user>/<model>[:quant]
	HFFileV                 string   `json:"hf_file_v,omitempty"`                  // -hffv, --hf-file-v FILE
	HFToken                 string   `json:"hf_token,omitempty"`                   // -hft, --hf-token TOKEN
	LogDisable              bool     `json:"log_disable,omitempty"`                // --log-disable
	LogFile                 string   `json:"log_file,omitempty"`                   // --log-file FNAME
	LogColors               string   `json:"log_colors,omitempty"`                 // --log-colors [on|off|auto]
	Verbose                 bool     `json:"verbose,omitempty"`                    // -v, --verbose, --log-verbose
	Offline                 bool     `json:"offline,omitempty"`                    // --offline
	Verbosity               int      `json:"verbosity,omitempty"`                  // -lv, --verbosity, --log-verbosity N
	LogPrefix               bool     `json:"log_prefix,omitempty"`                 // --log-prefix
	LogTimestamps           bool     `json:"log_timestamps,omitempty"`             // --log-timestamps
	CacheTypeKDraft         string   `json:"cache_type_k_draft,omitempty"`         // -ctkd, --cache-type-k-draft TYPE
	CacheTypeVDraft         string   `json:"cache_type_v_draft,omitempty"`         // -ctvd, --cache-type-v-draft TYPE

	// Sampling params
	Samplers         string   `json:"samplers,omitempty"`          // --samplers SAMPLERS
	Seed             int      `json:"seed,omitempty"`              // -s, --seed SEED
	SamplingSeq      string   `json:"sampling_seq,omitempty"`      // --sampler-seq, --sampling-seq SEQUENCE
	IgnoreEOS        bool     `json:"ignore_eos,omitempty"`        // --ignore-eos
	Temperature      float64  `json:"temp,omitempty"`              // --temp N
	TopK             int      `json:"top_k,omitempty"`             // --top-k N
	TopP             float64  `json:"top_p,omitempty"`             // --top-p N
	MinP             float64  `json:"min_p,omitempty"`             // --min-p N
	TopNSigma        float64  `json:"top_nsigma,omitempty"`        // --top-nsigma N
	XTCProbability   float64  `json:"xtc_probability,omitempty"`   // --xtc-probability N
	XTCThreshold     float64  `json:"xtc_threshold,omitempty"`     // --xtc-threshold N
	Typical          float64  `json:"typical,omitempty"`           // --typical N
	RepeatLastN      int      `json:"repeat_last_n,omitempty"`     // --repeat-last-n N
	RepeatPenalty    float64  `json:"repeat_penalty,omitempty"`    // --repeat-penalty N
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`  // --presence-penalty N
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"` // --frequency-penalty N
	DryMultiplier    float64  `json:"dry_multiplier,omitempty"`    // --dry-multiplier N
	DryBase          float64  `json:"dry_base,omitempty"`          // --dry-base N
	DryAllowedLength int      `json:"dry_allowed_length,omitempty"`
	DryPenaltyLastN  int      `json:"dry_penalty_last_n,omitempty"`
	DrySequenceBreaker []string `json:"dry_sequence_breaker,omitempty"` // --dry-sequence-breaker STRING
	AdaptiveTarget     float64  `json:"adaptive_target,omitempty"`      // --adaptive-target N
	AdaptiveDecay      float64  `json:"adaptive_decay,omitempty"`       // --adaptive-decay N
	DynatempRange      float64  `json:"dynatemp_range,omitempty"`       // --dynatemp-range N
	DynatempExp        float64  `json:"dynatemp_exp,omitempty"`         // --dynatemp-exp N
	Mirostat           int      `json:"mirostat,omitempty"`             // --mirostat N
	MirostatLR         float64  `json:"mirostat_lr,omitempty"`          // --mirostat-lr N
	MirostatEnt        float64  `json:"mirostat_ent,omitempty"`         // --mirostat-ent N
	LogitBias          []string `json:"logit_bias,omitempty"`           // -l, --logit-bias TOKEN_ID(+/-)BIAS
	Grammar            string   `json:"grammar,omitempty"`              // --grammar GRAMMAR
	GrammarFile        string   `json:"grammar_file,omitempty"`         // --grammar-file FNAME
	JSONSchema         string   `json:"json_schema,omitempty"`          // -j, --json-schema SCHEMA
	JSONSchemaFile     string   `json:"json_schema_file,omitempty"`     // -jf, --json-schema-file FILE
	BackendSampling    bool     `json:"backend_sampling,omitempty"`     // -bs, --backend-sampling

	// Server-specific params
	CtxCheckpoints       int      `json:"ctx_checkpoints,omitempty"`        // --ctx-checkpoints, --swa-checkpoints N
	CacheRAM             int      `json:"cache_ram,omitempty"`              // -cram, --cache-ram N
	LookupCacheStatic    bool     `json:"lookup_cache_static,omitempty"`    // -lcs, --lookup-cache-static
	LookupCacheDynamic   bool     `json:"lookup_cache_dynamic,omitempty"`   // -lcd, --lookup-cache-dynamic
	KVUnified            bool     `json:"kv_unified,omitempty"`             // -kvu, --kv-unified
	NoKVUnified          bool     `json:"no_kv_unified,omitempty"`          // -no-kvu, --no-kv-unified
	ContextShift         bool     `json:"context_shift,omitempty"`          // --context-shift
	NoContextShift       bool     `json:"no_context_shift,omitempty"`       // --no-context-shift
	ReversePrompt        string   `json:"reverse_prompt,omitempty"`         // -r, --reverse-prompt PROMPT
	Special              bool     `json:"special,omitempty"`                // -sp, --special
	Warmup               bool     `json:"warmup,omitempty"`                 // --warmup
	NoWarmup             bool     `json:"no_warmup,omitempty"`              // --no-warmup
	SPMInfill            bool     `json:"spm_infill,omitempty"`             // --spm-infill
	Pooling              string   `json:"pooling,omitempty"`                // --pooling {none,mean,cls,last,rank}
	Parallel             int      `json:"parallel,omitempty"`               // -np, --parallel N
	ContBatching         bool     `json:"cont_batching,omitempty"`          // -cb, --cont-batching
	NoContBatching       bool     `json:"no_cont_batching,omitempty"`       // -nocb, --no-cont-batching
	MMProj               string   `json:"mmproj,omitempty"`                 // -mm, --mmproj FILE
	MMProjURL            string   `json:"mmproj_url,omitempty"`             // -mmu, --mmproj-url URL
	MMProjAuto           bool     `json:"mmproj_auto,omitempty"`            // --mmproj-auto
	NoMMProj             bool     `json:"no_mmproj,omitempty"`              // --no-mmproj
	NoMMProjAuto         bool     `json:"no_mmproj_auto,omitempty"`         // --no-mmproj-auto
	MMProjOffload        bool     `json:"mmproj_offload,omitempty"`         // --mmproj-offload
	NoMMProjOffload      bool     `json:"no_mmproj_offload,omitempty"`      // --no-mmproj-offload
	ImageMinTokens       int      `json:"image_min_tokens,omitempty"`       // --image-min-tokens N
	ImageMaxTokens       int      `json:"image_max_tokens,omitempty"`       // --image-max-tokens N
	OverrideTensorDraft  []string `json:"override_tensor_draft,omitempty"`  // -otd, --override-tensor-draft
	CPUMoeDraft          bool     `json:"cpu_moe_draft,omitempty"`          // -cmoed, --cpu-moe-draft
	NCPUMoeDraft         int      `json:"n_cpu_moe_draft,omitempty"`        // -ncmoed, --n-cpu-moe-draft N
	Alias                string   `json:"alias,omitempty"`                  // -a, --alias STRING
	Host                 string   `json:"host,omitempty"`                   // --host HOST
	Port                 int      `json:"port,omitempty"`                   // --port PORT
	Path                 string   `json:"path,omitempty"`                   // --path PATH
	APIPrefix            string   `json:"api_prefix,omitempty"`             // --api-prefix PREFIX
	WebUIConfig          string   `json:"webui_config,omitempty"`           // --webui-config JSON
	WebUIConfigFile      string   `json:"webui_config_file,omitempty"`      // --webui-config-file PATH
	WebUI                bool     `json:"webui,omitempty"`                  // --webui
	NoWebUI              bool     `json:"no_webui,omitempty"`               // --no-webui
	Embedding            bool     `json:"embedding,omitempty"`              // --embedding, --embeddings
	Reranking            bool     `json:"reranking,omitempty"`              // --rerank, --reranking
	APIKey               string   `json:"api_key,omitempty"`                // --api-key KEY
	APIKeyFile           string   `json:"api_key_file,omitempty"`           // --api-key-file FNAME
	SSLKeyFile           string   `json:"ssl_key_file,omitempty"`           // --ssl-key-file FNAME
	SSLCertFile          string   `json:"ssl_cert_file,omitempty"`          // --ssl-cert-file FNAME
	ChatTemplateKwargs   string   `json:"chat_template_kwargs,omitempty"`   // --chat-template-kwargs STRING
	Timeout              int      `json:"timeout,omitempty"`                // -to, --timeout N
	ThreadsHTTP          int      `json:"threads_http,omitempty"`           // --threads-http N
	CachePrompt          bool     `json:"cache_prompt,omitempty"`           // --cache-prompt
	NoCachePrompt        bool     `json:"no_cache_prompt,omitempty"`        // --no-cache-prompt
	CacheReuse           int      `json:"cache_reuse,omitempty"`            // --cache-reuse N
	Metrics              bool     `json:"metrics,omitempty"`                // --metrics
	Props                bool     `json:"props,omitempty"`                  // --props
	Slots                bool     `json:"slots,omitempty"`                  // --slots
	NoSlots              bool     `json:"no_slots,omitempty"`               // --no-slots
	SlotSavePath         string   `json:"slot_save_path,omitempty"`         // --slot-save-path PATH
	MediaPath            string   `json:"media_path,omitempty"`             // --media-path PATH
	ModelsDir            string   `json:"models_dir,omitempty"`             // --models-dir PATH
	ModelsPreset         string   `json:"models_preset,omitempty"`          // --models-preset PATH
	ModelsMax            int      `json:"models_max,omitempty"`             // --models-max N
	ModelsAutoload       bool     `json:"models_autoload,omitempty"`        // --models-autoload
	NoModelsAutoload     bool     `json:"no_models_autoload,omitempty"`     // --no-models-autoload
	Jinja                bool     `json:"jinja,omitempty"`                  // --jinja
	NoJinja              bool     `json:"no_jinja,omitempty"`               // --no-jinja
	ReasoningFormat      string   `json:"reasoning_format,omitempty"`       // --reasoning-format FORMAT
	ReasoningBudget      int      `json:"reasoning_budget,omitempty"`       // --reasoning-budget N
	ChatTemplate         string   `json:"chat_template,omitempty"`          // --chat-template JINJA_TEMPLATE
	ChatTemplateFile     string   `json:"chat_template_file,omitempty"`     // --chat-template-file JINJA_TEMPLATE_FILE
	PrefillAssistant     bool     `json:"prefill_assistant,omitempty"`      // --prefill-assistant
	NoPrefillAssistant   bool     `json:"no_prefill_assistant,omitempty"`   // --no-prefill-assistant
	SlotPromptSimilarity float64  `json:"slot_prompt_similarity,omitempty"` // -sps, --slot-prompt-similarity SIMILARITY
	LoraInitWithoutApply bool     `json:"lora_init_without_apply,omitempty"`
	SleepIdleSeconds     int      `json:"sleep_idle_seconds,omitempty"` // --sleep-idle-seconds SECONDS
	ThreadsDraft         int      `json:"threads_draft,omitempty"`      // -td, --threads-draft N
	ThreadsBatchDraft    int      `json:"threads_batch_draft,omitempty"` // -tbd, --threads-batch-draft N
	DraftMax             int      `json:"draft_max,omitempty"`          // --draft, --draft-n, --draft-max N
	DraftMin             int      `json:"draft_min,omitempty"`          // --draft-min, --draft-n-min N
	DraftPMin            float64  `json:"draft_p_min,omitempty"`        // --draft-p-min P
	CtxSizeDraft         int      `json:"ctx_size_draft,omitempty"`     // -cd, --ctx-size-draft N
	DeviceDraft          string   `json:"device_draft,omitempty"`       // -devd, --device-draft <dev1,dev2,..>
	GPULayersDraft       int      `json:"gpu_layers_draft,omitempty"`   // -ngld, --gpu-layers-draft, --n-gpu-layers-draft N
	ModelDraft           string   `json:"model_draft,omitempty"`        // -md, --model-draft FNAME
	SpecReplace          string   `json:"spec_replace,omitempty"`       // --spec-replace TARGET DRAFT
	SpecType             string   `json:"spec_type,omitempty"`          // --spec-type TYPE
	SpecNgramSizeN       int      `json:"spec_ngram_size_n,omitempty"`  // --spec-ngram-size-n N
	SpecNgramSizeM       int      `json:"spec_ngram_size_m,omitempty"`  // --spec-ngram-size-m M
	SpecNgramCheckRate   float64  `json:"spec_ngram_check_rate,omitempty"` // --spec-ngram-check-rate RATE
	SpecNgramMinHits     int      `json:"spec_ngram_min_hits,omitempty"` // --spec-ngram-min-hits N
	ModelVocoder         string   `json:"model_vocoder,omitempty"`      // -mv, --model-vocoder FNAME
	TTSUseGuideTokens    bool     `json:"tts_use_guide_tokens,omitempty"`

	// Default model params
	EmbdGemmaDefault      bool `json:"embd_gemma_default,omitempty"`       // --embd-gemma-default
	FIMQwen1B             bool `json:"fim_qwen_1b,omitempty"`              // --fim-qwen-1
	FIMQwen1_5BDefault    bool `json:"fim_qwen_1_5b_default,omitempty"`    // --fim-qwen-1.5b-default
	FIMQwen3BDefault      bool `json:"fim_qwen_3b_default,omitempty"`      // --fim-qwen-3b-default
	FIMQwen7BDefault      bool `json:"fim_qwen_7b_default,omitempty"`      // --fim-qwen-7b-default
	FIMQwen7BSpec         bool `json:"fim_qwen_7b_spec,omitempty"`         // --fim-qwen-7b-spec
	FIMQwen14BSpec        bool `json:"fim_qwen_14b_spec,omitempty"`        // --fim-qwen-14b-spec
	FIMQwen30BDefault     bool `json:"fim_qwen_30b_default,omitempty"`     // --fim-qwen-30b-default
	GPTOss20BDefault      bool `json:"gpt_oss_20b_default,omitempty"`      // --gpt-oss-20b-default
	GPTOss120BDefault     bool `json:"gpt_oss_120b_default,omitempty"`     // --gpt-oss-120b-default
	VisionGemma4BDefault  bool `json:"vision_gemma_4b_default,omitempty"`  // --vision-gemma-4b-default
	VisionGemma12BDefault bool `json:"vision_gemma_12b_default,omitempty"` // --vision-gemma-12b-default

	// ExtraArgs are additional command line arguments.
	// Example: {"verbose": "", "log-file": "/logs/llama.log"}
	ExtraArgs map[string]string `json:"extra_args,omitempty"`
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

	// Track which fields we've processed
	processedFields := make(map[string]bool)

	// Get all known canonical field names from struct tags
	knownFields := getKnownFieldNames(o)
	for field := range knownFields {
		processedFields[field] = true
	}

	// Process alternative field names and mark them as processed
	for altName, canonicalName := range llamaFieldMappings {
		processedFields[altName] = true // Mark alternatives as known

		if value, exists := raw[altName]; exists {
			// Use reflection to set the field value
			v := reflect.ValueOf(o).Elem()
			field := v.FieldByNameFunc(func(fieldName string) bool {
				field, _ := v.Type().FieldByName(fieldName)
				jsonTag := field.Tag.Get("json")
				return jsonTag == canonicalName+",omitempty" || jsonTag == canonicalName
			})

			if field.IsValid() && field.CanSet() {
				setFieldValue(field, value)
			}
		}
	}

	// Collect unknown fields into ExtraArgs
	if o.ExtraArgs == nil {
		o.ExtraArgs = make(map[string]string)
	}
	for key, value := range raw {
		if !processedFields[key] {
			o.ExtraArgs[key] = fmt.Sprintf("%v", value)
		}
	}

	return nil
}

func (o *LlamaServerOptions) GetModel() string {
	if o.Model != "" {
		return o.Model
	} else if o.HFRepo != "" {
		return o.HFRepo
	}
	return ""
}

func (o *LlamaServerOptions) GetPort() int {
	if o == nil {
		return 0
	}
	return o.Port
}

func (o *LlamaServerOptions) SetPort(port int) {
	if o == nil {
		return
	}
	o.Port = port
}

func (o *LlamaServerOptions) GetHost() string {
	if o == nil {
		return "localhost"
	}
	return o.Host
}

func (o *LlamaServerOptions) Validate() error {
	// Allow nil options for router mode where llama.cpp manages models dynamically
	if o == nil {
		return nil
	}

	// Use reflection to check all string fields for injection patterns
	if err := validation.ValidateStructStrings(o, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if o.Port < 0 || o.Port > 65535 {
		return validation.ValidationError(fmt.Errorf("invalid port range: %d", o.Port))
	}

	// Validate extra_args keys and values
	for key, value := range o.ExtraArgs {
		if err := validation.ValidateStringForInjection(key); err != nil {
			return validation.ValidationError(fmt.Errorf("extra_args key %q: %w", key, err))
		}
		if value != "" {
			if err := validation.ValidateStringForInjection(value); err != nil {
				return validation.ValidationError(fmt.Errorf("extra_args value for %q: %w", key, err))
			}
		}
	}

	return nil
}

// BuildCommandArgs converts InstanceOptions to command line arguments
func (o *LlamaServerOptions) BuildCommandArgs() []string {
	if o == nil {
		return []string{}
	}
	// Llama uses multiple flags for arrays by default (not comma-separated)
	// Use package-level llamaMultiValuedFlags variable
	args := BuildCommandArgs(o, llamaMultiValuedFlags)

	// Append extra args at the end
	args = append(args, convertExtraArgsToFlags(o.ExtraArgs)...)

	return args
}

func (o *LlamaServerOptions) BuildDockerArgs() []string {
	if o == nil {
		return []string{}
	}
	// For llama, Docker args are the same as normal args
	return o.BuildCommandArgs()
}

// llamaFieldMappings maps alternative field names (short forms, aliases) to canonical snake_case names
// Used for both JSON unmarshaling and command-line parsing
var llamaFieldMappings = map[string]string{
	// Common params
	"t":              "threads",         // -t, --threads N
	"tb":             "threads_batch",   // -tb, --threads-batch N
	"C":              "cpu_mask",        // -C, --cpu-mask M
	"Cr":             "cpu_range",       // -Cr, --cpu-range lo-hi
	"Cb":             "cpu_mask_batch",  // -Cb, --cpu-mask-batch M
	"Crb":            "cpu_range_batch", // -Crb, --cpu-range-batch lo-hi
	"c":              "ctx_size",        // -c, --ctx-size N
	"n":              "predict",         // -n, --predict N
	"n_predict":      "predict",         // --n-predict N
	"b":              "batch_size",      // -b, --batch-size N
	"ub":             "ubatch_size",     // -ub, --ubatch-size N
	"fa":             "flash_attn",      // -fa, --flash-attn
	"e":              "escape",          // -e, --escape
	"kvo":            "kv_offload",      // -kvo, --kv-offload
	"nkvo":           "no_kv_offload",   // -nkvo, --no-kv-offload
	"nr":             "no_repack",       // -nr, --no-repack
	"ctk":            "cache_type_k",    // -ctk, --cache-type-k TYPE
	"ctv":            "cache_type_v",    // -ctv, --cache-type-v TYPE
	"dt":             "defrag_thold",    // -dt, --defrag-thold N
	"dio":            "direct_io",       // -dio, --direct-io
	"ndio":           "no_direct_io",    // -ndio, --no-direct-io
	"dev":            "device",          // -dev, --device <dev1,dev2,..>
	"ot":             "override_tensor", // -ot, --override-tensor
	"cmoe":           "cpu_moe",         // -cmoe, --cpu-moe
	"ncmoe":          "n_cpu_moe",       // -ncmoe, --n-cpu-moe N
	"ngl":            "gpu_layers",      // -ngl, --gpu-layers N
	"n_gpu_layers":   "gpu_layers",      // --n-gpu-layers N
	"sm":             "split_mode",      // -sm, --split-mode
	"ts":             "tensor_split",    // -ts, --tensor-split N0,N1,N2,...
	"mg":             "main_gpu",        // -mg, --main-gpu INDEX
	"fitt":           "fit_target",      // -fitt, --fit-target MiB0,MiB1,MiB2,...
	"fitc":           "fit_ctx",         // -fitc, --fit-ctx N
	"m":              "model",           // -m, --model FNAME
	"mu":             "model_url",       // -mu, --model-url MODEL_URL
	"dr":             "docker_repo",     // -dr, --docker-repo
	"hf":             "hf_repo",         // -hf, --hf-repo
	"hfr":            "hf_repo",         // -hfr, --hf-repo
	"hfd":            "hf_repo_draft",   // -hfd, --hf-repo-draft
	"hfrd":           "hf_repo_draft",   // -hfrd, --hf-repo-draft
	"hff":            "hf_file",         // -hff, --hf-file FILE
	"hfv":            "hf_repo_v",       // -hfv, --hf-repo-v
	"hfrv":           "hf_repo_v",       // -hfrv, --hf-repo-v
	"hffv":           "hf_file_v",       // -hffv, --hf-file-v FILE
	"hft":            "hf_token",        // -hft, --hf-token TOKEN
	"v":              "verbose",         // -v, --verbose
	"log_verbose":    "verbose",         // --log-verbose
	"lv":             "verbosity",       // -lv, --verbosity
	"log_verbosity":  "verbosity",       // --log-verbosity N
	"ctkd":           "cache_type_k_draft", // -ctkd, --cache-type-k-draft TYPE
	"ctvd":           "cache_type_v_draft", // -ctvd, --cache-type-v-draft TYPE

	// Sampling params
	"s":           "seed",             // -s, --seed SEED
	"sampler_seq": "sampling_seq",     // --sampler-seq, --sampling-seq
	"l":           "logit_bias",       // -l, --logit-bias
	"j":           "json_schema",      // -j, --json-schema SCHEMA
	"jf":          "json_schema_file", // -jf, --json-schema-file FILE
	"bs":          "backend_sampling", // -bs, --backend-sampling

	// Server-specific params
	"swa_checkpoints":    "ctx_checkpoints",        // --swa-checkpoints N
	"cram":               "cache_ram",              // -cram, --cache-ram N
	"lcs":                "lookup_cache_static",    // -lcs, --lookup-cache-static
	"lcd":                "lookup_cache_dynamic",   // -lcd, --lookup-cache-dynamic
	"kvu":                "kv_unified",             // -kvu, --kv-unified
	"no_kvu":             "no_kv_unified",          // -no-kvu, --no-kv-unified
	"r":                  "reverse_prompt",         // -r, --reverse-prompt
	"sp":                 "special",                // -sp, --special
	"np":                 "parallel",               // -np, --parallel N
	"cb":                 "cont_batching",          // -cb, --cont-batching
	"nocb":               "no_cont_batching",       // -nocb, --no-cont-batching
	"mm":                 "mmproj",                 // -mm, --mmproj FILE
	"mmu":                "mmproj_url",             // -mmu, --mmproj-url URL
	"no_mmproj":          "no_mmproj_auto",         // --no-mmproj (alias for --no-mmproj-auto)
	"otd":                "override_tensor_draft",  // -otd, --override-tensor-draft
	"cmoed":              "cpu_moe_draft",          // -cmoed, --cpu-moe-draft
	"ncmoed":             "n_cpu_moe_draft",        // -ncmoed, --n-cpu-moe-draft N
	"a":                  "alias",                  // -a, --alias STRING
	"embeddings":         "embedding",              // --embeddings
	"rerank":             "reranking",              // --reranking
	"to":                 "timeout",                // -to, --timeout N
	"sps":                "slot_prompt_similarity", // -sps, --slot-prompt-similarity
	"td":                 "threads_draft",          // -td, --threads-draft N
	"tbd":                "threads_batch_draft",    // -tbd, --threads-batch-draft N
	"draft":              "draft_max",              // --draft, --draft-max N
	"draft_n":            "draft_max",              // --draft-n N
	"draft_min":          "draft_min",              // --draft-min N
	"draft_n_min":        "draft_min",              // --draft-n-min N
	"cd":                 "ctx_size_draft",         // -cd, --ctx-size-draft N
	"devd":               "device_draft",           // -devd, --device-draft
	"ngld":               "gpu_layers_draft",       // -ngld, --gpu-layers-draft N
	"n_gpu_layers_draft": "gpu_layers_draft",       // --n-gpu-layers-draft N
	"md":                 "model_draft",            // -md, --model-draft FNAME
	"mv":                 "model_vocoder",          // -mv, --model-vocoder FNAME
	"fim_qwen_1":         "fim_qwen_1b",            // --fim-qwen-1 → fim_qwen_1b
}

// ParseCommand parses a llama-server command string into LlamaServerOptions
// Supports multiple formats:
// 1. Full command: "llama-server --model file.gguf"
// 2. Full path: "/usr/local/bin/llama-server --model file.gguf"
// 3. Args only: "--model file.gguf --gpu-layers 32"
// 4. Multiline commands with backslashes
func (o *LlamaServerOptions) ParseCommand(command string) (any, error) {
	executableNames := []string{"llama-server"}
	var subcommandNames []string // Llama has no subcommands

	var llamaOptions LlamaServerOptions
	if err := parseCommandWithAliases(command, executableNames, subcommandNames, llamaMultiValuedFlags, llamaFieldMappings, &llamaOptions); err != nil {
		return nil, err
	}

	return &llamaOptions, nil
}
