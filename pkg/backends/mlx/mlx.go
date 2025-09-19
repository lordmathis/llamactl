package mlx

import (
	"encoding/json"
	"llamactl/pkg/backends"
	"reflect"
	"strconv"
)

type MlxServerOptions struct {
	// Basic connection options
	Model string `json:"model,omitempty"`
	Host  string `json:"host,omitempty"`
	Port  int    `json:"port,omitempty"`

	// Model and adapter options
	AdapterPath     string `json:"adapter_path,omitempty"`
	DraftModel      string `json:"draft_model,omitempty"`
	NumDraftTokens  int    `json:"num_draft_tokens,omitempty"`
	TrustRemoteCode bool   `json:"trust_remote_code,omitempty"`

	// Logging and templates
	LogLevel               string `json:"log_level,omitempty"`
	ChatTemplate           string `json:"chat_template,omitempty"`
	UseDefaultChatTemplate bool   `json:"use_default_chat_template,omitempty"`
	ChatTemplateArgs       string `json:"chat_template_args,omitempty"` // JSON string

	// Sampling defaults
	Temp      float64 `json:"temp,omitempty"` // Note: MLX uses "temp" not "temperature"
	TopP      float64 `json:"top_p,omitempty"`
	TopK      int     `json:"top_k,omitempty"`
	MinP      float64 `json:"min_p,omitempty"`
	MaxTokens int     `json:"max_tokens,omitempty"`
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
		"m":           "model",       // -m, --model
		"temperature": "temp",        // --temperature vs --temp
		"top_k":       "top_k",       // --top-k
		"adapter_path": "adapter_path", // --adapter-path
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

// BuildCommandArgs converts to command line arguments
func (o *MlxServerOptions) BuildCommandArgs() []string {
	multipleFlags := map[string]bool{} // MLX doesn't currently have []string fields
	return backends.BuildCommandArgs(o, multipleFlags)
}
