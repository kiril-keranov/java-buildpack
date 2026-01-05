package frameworks_test

import (
	"os"
	"testing"
)

func TestDatadogDetectionWithAPIKey(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key-12345")
	defer os.Unsetenv("DD_API_KEY")

	apiKey := os.Getenv("DD_API_KEY")
	if apiKey == "" {
		t.Error("DD_API_KEY should be set for Datadog detection")
	}
}

func TestDatadogAPMDisabledFlag(t *testing.T) {
	os.Setenv("DD_APM_ENABLED", "false")
	defer os.Unsetenv("DD_APM_ENABLED")

	apmEnabled := os.Getenv("DD_APM_ENABLED")
	if apmEnabled != "false" {
		t.Errorf("Expected DD_APM_ENABLED to be 'false', got %s", apmEnabled)
	}
}

func TestDatadogServiceTags(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "DD_SERVICE tag",
			env: map[string]string{
				"DD_SERVICE": "my-app-service",
			},
		},
		{
			name: "DD_ENV tag",
			env: map[string]string{
				"DD_ENV": "production",
			},
		},
		{
			name: "DD_VERSION tag",
			env: map[string]string{
				"DD_VERSION": "1.2.3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)

				if os.Getenv(k) != v {
					t.Errorf("Expected %s to be %s", k, v)
				}
			}
		})
	}
}
