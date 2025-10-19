package backends

type BackendType string

const (
	BackendTypeLlamaCpp BackendType = "llama_cpp"
	BackendTypeMlxLm    BackendType = "mlx_lm"
	BackendTypeVllm     BackendType = "vllm"
	// BackendTypeMlxVlm BackendType = "mlx_vlm"  // Future expansion
)

type Options struct {
	BackendType    BackendType    `json:"backend_type"`
	BackendOptions map[string]any `json:"backend_options,omitempty"`

	Nodes map[string]struct{} `json:"-"`

	// Backend-specific options
	LlamaServerOptions *LlamaServerOptions `json:"-"`
	MlxServerOptions   *MlxServerOptions   `json:"-"`
	VllmServerOptions  *VllmServerOptions  `json:"-"`
}
