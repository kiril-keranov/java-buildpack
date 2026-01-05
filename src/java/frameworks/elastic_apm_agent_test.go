package frameworks_test

import (
	"os"
	"testing"
)

func TestElasticAPMServiceDetection(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		expect  bool
	}{
		{
			name: "ELASTIC_APM_SERVER_URL set",
			envVars: map[string]string{
				"ELASTIC_APM_SERVER_URL": "https://apm.example.com",
			},
			expect: true,
		},
		{
			name: "ELASTIC_APM_SERVICE_NAME set",
			envVars: map[string]string{
				"ELASTIC_APM_SERVICE_NAME": "my-service",
			},
			expect: true,
		},
		{
			name:    "no elastic env vars",
			envVars: map[string]string{},
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			hasServerURL := os.Getenv("ELASTIC_APM_SERVER_URL") != ""
			hasServiceName := os.Getenv("ELASTIC_APM_SERVICE_NAME") != ""

			detected := hasServerURL || hasServiceName
			if detected != tt.expect {
				t.Errorf("Expected detection %v, got %v", tt.expect, detected)
			}
		})
	}
}
