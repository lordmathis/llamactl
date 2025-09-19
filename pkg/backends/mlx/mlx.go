package mlx

import (
	"reflect"
	"strconv"
	"strings"
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

// BuildCommandArgs converts to command line arguments using reflection
func (o *MlxServerOptions) BuildCommandArgs() []string {
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
		}
	}

	return args
}
