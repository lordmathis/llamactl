package manager

import (
	"testing"
)

// TestValidateInstanceNameForURL tests the URL validation function
func TestValidateInstanceNameForURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		// Valid names
		{"valid simple name", "test-instance", false},
		{"valid with underscores", "my_instance", false},
		{"valid with numbers", "instance123", false},
		{"valid with dashes", "test-name-with-dashes", false},

		// Invalid names - path traversal
		{"path traversal with ..", "../../etc/passwd", true},
		{"path traversal multiple", "../../../etc/shadow", true},
		{"path traversal in middle", "foo/../bar", true},
		{"double dots variation", ".../", true},

		// Invalid names - path separators
		{"forward slash", "foo/bar", true},
		{"backslash", "foo\\bar", true},
		{"absolute path", "/etc/passwd", true},

		// Invalid names - URL-unsafe characters
		{"question mark", "test?param=value", true},
		{"ampersand", "test&param", true},
		{"hash", "test#anchor", true},
		{"percent", "test%20space", true},
		{"equals", "test=value", true},
		{"at sign", "test@example", true},
		{"colon", "test:8080", true},
		{"space", "test instance", true},

		// Invalid names - empty
		{"empty string", "", true},

		// Invalid names - characters requiring encoding
		{"unicode", "test\u00e9", true},
		{"newline", "test\n", true},
		{"tab", "test\t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedName, err := validateInstanceNameForURL(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for input %q, but got: %v", tt.input, err)
				}
				if validatedName != tt.input {
					t.Errorf("Expected validated name to be %q, but got %q", tt.input, validatedName)
				}
			}
		})
	}
}
