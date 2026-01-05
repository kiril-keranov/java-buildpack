package frameworks_test

import (
	"os"
	"testing"
)

func TestSkyWalkingConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "SW_AGENT_COLLECTOR_BACKEND_SERVICES",
			env:   "SW_AGENT_COLLECTOR_BACKEND_SERVICES",
			value: "skywalking-oap.example.com:11800",
		},
		{
			name:  "SW_AGENT_NAME",
			env:   "SW_AGENT_NAME",
			value: "my-app",
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
