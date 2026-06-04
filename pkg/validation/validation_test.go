package validation_test

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/validation"
	"strings"
	"testing"
)

func TestValidateInstanceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple name", "myinstance", false},
		{"with numbers", "instance123", false},
		{"with hyphens", "my-instance", false},
		{"with underscores", "my_instance", false},
		{"with dots", "my.instance", false},
		{"mixed valid chars", "test-instance_123", false},
		{"single char", "a", false},
		{"max length", strings.Repeat("a", 50), false},

		{"empty name", "", true},
		{"with spaces", "my instance", true},
		{"with special chars", "my@instance", true},
		{"too long", strings.Repeat("a", 51), true},
		{"shell metachar semicolon", "test;ls", true},
		{"shell metachar pipe", "test|ls", true},
		{"shell metachar ampersand", "test&ls", true},
		{"shell metachar dollar", "test$var", true},
		{"shell metachar backtick", "test`cmd`", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := validation.ValidateInstanceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if name != tt.input {
				t.Errorf("ValidateInstanceName(%q) = %q, want %q", tt.input, name, tt.input)
			}
		})
	}
}

func TestValidateInstanceOptions_NilOptions(t *testing.T) {
	var opts backends.Options
	err := opts.ValidateInstanceOptions()
	if err == nil {
		t.Error("Expected error for nil options")
	}
}

func TestValidateInstanceOptions_PortValidation(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 0", 0, false},
		{"valid port 80", 80, false},
		{"valid port 8080", 8080, false},
		{"valid port 65535", 65535, false},
		{"negative port", -1, true},
		{"port too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Port: tt.port,
				},
			}

			err := options.ValidateInstanceOptions()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceOptions(port=%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}
