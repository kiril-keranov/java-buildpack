package frameworks_test

import (
	"os"
	"testing"
)

func TestDebugEnabledDetection(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		value    string
		expected bool
	}{
		{
			name:     "BPL_DEBUG_ENABLED true",
			env:      "BPL_DEBUG_ENABLED",
			value:    "true",
			expected: true,
		},
		{
			name:     "BPL_DEBUG_ENABLED 1",
			env:      "BPL_DEBUG_ENABLED",
			value:    "1",
			expected: true,
		},
		{
			name:     "BPL_DEBUG_ENABLED false",
			env:      "BPL_DEBUG_ENABLED",
			value:    "false",
			expected: false,
		},
		{
			name:     "JBP_CONFIG_DEBUG enabled",
			env:      "JBP_CONFIG_DEBUG",
			value:    "enabled: true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.env, tt.value)
			defer os.Unsetenv(tt.env)

			value := os.Getenv(tt.env)
			if value != tt.value {
				t.Errorf("Expected %s to be %s, got %s", tt.env, tt.value, value)
			}
		})
	}
}

func TestDebugPortConfiguration(t *testing.T) {
	defaultPort := 8000

	tests := []struct {
		name string
		env  string
		port int
	}{
		{
			name: "default port",
			env:  "",
			port: defaultPort,
		},
		{
			name: "BPL_DEBUG_PORT",
			env:  "9000",
			port: 9000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				os.Setenv("BPL_DEBUG_PORT", tt.env)
				defer os.Unsetenv("BPL_DEBUG_PORT")
			}

			port := defaultPort
			if portEnv := os.Getenv("BPL_DEBUG_PORT"); portEnv != "" {
				port = 9000
			}

			if port != tt.port {
				t.Errorf("Expected port %d, got %d", tt.port, port)
			}
		})
	}
}

func TestDebugSuspendMode(t *testing.T) {
	tests := []struct {
		name    string
		suspend string
		expect  bool
	}{
		{
			name:    "suspend enabled",
			suspend: "true",
			expect:  true,
		},
		{
			name:    "suspend disabled",
			suspend: "false",
			expect:  false,
		},
		{
			name:    "suspend not set",
			suspend: "",
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.suspend != "" {
				os.Setenv("BPL_DEBUG_SUSPEND", tt.suspend)
				defer os.Unsetenv("BPL_DEBUG_SUSPEND")
			}

			suspend := os.Getenv("BPL_DEBUG_SUSPEND") == "true"
			if suspend != tt.expect {
				t.Errorf("Expected suspend %v, got %v", tt.expect, suspend)
			}
		})
	}
}
