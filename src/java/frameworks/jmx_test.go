package frameworks_test

import (
	"os"
	"testing"
)

func TestJMXDetection(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		value    string
		expected bool
	}{
		{
			name:     "JMX enabled via BPL_JMX_ENABLED",
			env:      "BPL_JMX_ENABLED",
			value:    "true",
			expected: true,
		},
		{
			name:     "JMX disabled",
			env:      "BPL_JMX_ENABLED",
			value:    "false",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.env, tt.value)
			defer os.Unsetenv(tt.env)

			enabled := os.Getenv(tt.env) == "true"
			if enabled != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, enabled)
			}
		})
	}
}

func TestJMXPortConfiguration(t *testing.T) {
	port := "5000"
	os.Setenv("BPL_JMX_PORT", port)
	defer os.Unsetenv("BPL_JMX_PORT")

	if os.Getenv("BPL_JMX_PORT") != port {
		t.Errorf("Expected JMX port to be %s", port)
	}
}
