package frameworks_test

import (
	"os"
	"testing"
)

func TestSplunkOtelConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "SPLUNK_ACCESS_TOKEN",
			env:   "SPLUNK_ACCESS_TOKEN",
			value: "test-token-abc123",
		},
		{
			name:  "SPLUNK_REALM",
			env:   "SPLUNK_REALM",
			value: "us0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.env, tt.value)
			defer os.Unsetenv(tt.env)

			if os.Getenv(tt.env) != tt.value {
				t.Errorf("Expected %s to be %s", tt.env, tt.value)
			}
		})
	}
}
