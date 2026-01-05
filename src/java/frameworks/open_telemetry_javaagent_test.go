package frameworks_test

import (
	"os"
	"testing"
)

func TestOpenTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "OTEL_SERVICE_NAME",
			env:   "OTEL_SERVICE_NAME",
			value: "my-service",
		},
		{
			name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			env:   "OTEL_EXPORTER_OTLP_ENDPOINT",
			value: "https://otel-collector.example.com",
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
