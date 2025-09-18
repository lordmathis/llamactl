package backends

type BackendType string

const (
	BackendTypeLlamaCpp BackendType = "llama_cpp"
	BackendTypeMlxLm    BackendType = "mlx_lm"
	// BackendTypeMlxVlm BackendType = "mlx_vlm"  // Future expansion
)
