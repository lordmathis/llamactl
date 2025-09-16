package mlx

import (
	"encoding/json"
	"reflect"
	"strconv"
)

type MlxServerOptions struct {
	// Basic connection options
	Model       string `json:"model,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
	PythonPath  string `json:"python_path,omitempty"`  // Custom: Python venv path
	
	// Model and adapter options
	AdapterPath     string `json:"adapter_path,omitempty"`
	DraftModel      string `json:"draft_model,omitempty"`
	NumDraftTokens  int    `json:"num_draft_tokens,omitempty"`
	TrustRemoteCode bool   `json:"trust_remote_code,omitempty"`
	
	// Logging and templates
	LogLevel                 string `json:"log_level,omitempty"`
	ChatTemplate             string `json:"chat_template,omitempty"`
	UseDefaultChatTemplate   bool   `json:"use_default_chat_template,omitempty"`
	ChatTemplateArgs         string `json:"chat_template_args,omitempty"` // JSON string
	
	// Sampling defaults
	Temp     float64 `json:"temp,omitempty"`      // Note: MLX uses "temp" not "temperature"
	TopP     float64 `json:"top_p,omitempty"`
	TopK     int     `json:"top_k,omitempty"`
	MinP     float64 `json:"min_p,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to support multiple field names
func (o *MlxServerOptions) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to handle multiple field names
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create a temporary struct for standard unmarshaling
	type tempOptions MlxServerOptions
	temp := tempOptions{}

	// Standard unmarshal first
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy to our struct
	*o = MlxServerOptions(temp)

	// Handle alternative field names
	fieldMappings := map[string]string{
		// Basic connection options
		"m":            "model",
		"host":         "host",
		"port":         "port",
		"python_path":  "python_path",
		
		// Model and adapter options
		"adapter-path":      "adapter_path",
		"draft-model":       "draft_model",
		"num-draft-tokens":  "num_draft_tokens",
		"trust-remote-code": "trust_remote_code",
		
		// Logging and templates
		"log-level":                   "log_level",
		"chat-template":               "chat_template",
		"use-default-chat-template":   "use_default_chat_template",
		"chat-template-args":          "chat_template_args",
		
		// Sampling defaults
		"temperature": "temp",        // Support both temp and temperature
		"top-p":       "top_p",
		"top-k":       "top_k",
		"min-p":       "min_p",
		"max-tokens":  "max_tokens",
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

// NewMlxServerOptions creates MlxServerOptions with MLX defaults
func NewMlxServerOptions() *MlxServerOptions {
	return &MlxServerOptions{
		Host:           "127.0.0.1",  // MLX default (different from llama-server)
		Port:           8080,         // MLX default
		NumDraftTokens: 3,            // MLX default for speculative decoding
		LogLevel:       "INFO",       // MLX default
		Temp:           0.0,          // MLX default
		TopP:           1.0,          // MLX default  
		TopK:           0,            // MLX default (disabled)
		MinP:           0.0,          // MLX default (disabled)
		MaxTokens:      512,          // MLX default
		ChatTemplateArgs: "{}",       // MLX default (empty JSON object)
	}
}

// BuildCommandArgs converts to command line arguments
func (o *MlxServerOptions) BuildCommandArgs() []string {
	var args []string
	
	// Note: PythonPath is handled in lifecycle.go execution logic
	
	// Required and basic options
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.Host != "" {
		args = append(args, "--host", o.Host)
	}
	if o.Port != 0 {
		args = append(args, "--port", strconv.Itoa(o.Port))
	}
	
	// Model and adapter options
	if o.AdapterPath != "" {
		args = append(args, "--adapter-path", o.AdapterPath)
	}
	if o.DraftModel != "" {
		args = append(args, "--draft-model", o.DraftModel)
	}
	if o.NumDraftTokens != 0 {
		args = append(args, "--num-draft-tokens", strconv.Itoa(o.NumDraftTokens))
	}
	if o.TrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}
	
	// Logging and templates
	if o.LogLevel != "" {
		args = append(args, "--log-level", o.LogLevel)
	}
	if o.ChatTemplate != "" {
		args = append(args, "--chat-template", o.ChatTemplate)
	}
	if o.UseDefaultChatTemplate {
		args = append(args, "--use-default-chat-template")
	}
	if o.ChatTemplateArgs != "" {
		args = append(args, "--chat-template-args", o.ChatTemplateArgs)
	}
	
	// Sampling defaults
	if o.Temp != 0 {
		args = append(args, "--temp", strconv.FormatFloat(o.Temp, 'f', -1, 64))
	}
	if o.TopP != 0 {
		args = append(args, "--top-p", strconv.FormatFloat(o.TopP, 'f', -1, 64))
	}
	if o.TopK != 0 {
		args = append(args, "--top-k", strconv.Itoa(o.TopK))
	}
	if o.MinP != 0 {
		args = append(args, "--min-p", strconv.FormatFloat(o.MinP, 'f', -1, 64))
	}
	if o.MaxTokens != 0 {
		args = append(args, "--max-tokens", strconv.Itoa(o.MaxTokens))
	}
	
	return args
}