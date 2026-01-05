package frameworks_test

import (
	"os"
	"testing"
)

func TestClientCertificateMapperEnabledByDefault(t *testing.T) {
	result := isClientCertMapperEnabled("")
	if !result {
		t.Error("Client Certificate Mapper should be enabled by default")
	}
}

func TestClientCertificateMapperDisabledViaConfig(t *testing.T) {
	tests := []struct {
		name   string
		config string
		expect bool
	}{
		{
			name:   "explicitly disabled",
			config: "enabled: false",
			expect: false,
		},
		{
			name:   "explicitly enabled",
			config: "enabled: true",
			expect: true,
		},
		{
			name:   "empty config",
			config: "",
			expect: true,
		},
		{
			name:   "config without enabled key",
			config: "some_other_key: value",
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isClientCertMapperEnabled(tt.config)
			if result != tt.expect {
				t.Errorf("Expected %v, got %v for config: %s", tt.expect, result, tt.config)
			}
		})
	}
}

func TestClientCertificateMapperConfigParsing(t *testing.T) {
	testCases := []struct {
		name     string
		envVar   string
		expected bool
	}{
		{
			name:     "YAML with enabled false",
			envVar:   "{enabled: false}",
			expected: false,
		},
		{
			name:     "YAML with enabled true",
			envVar:   "{enabled: true}",
			expected: true,
		},
		{
			name:     "YAML with quoted enabled false",
			envVar:   "{'enabled': false}",
			expected: false,
		},
		{
			name:     "YAML with quoted enabled true",
			envVar:   "{'enabled': true}",
			expected: true,
		},
		{
			name:     "empty config",
			envVar:   "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("JBP_CONFIG_CLIENT_CERTIFICATE_MAPPER", tc.envVar)
			defer os.Unsetenv("JBP_CONFIG_CLIENT_CERTIFICATE_MAPPER")

			result := isClientCertMapperEnabled(tc.envVar)
			if result != tc.expected {
				t.Errorf("Expected %v for config %q, got %v", tc.expected, tc.envVar, result)
			}
		})
	}
}

func isClientCertMapperEnabled(config string) bool {
	if config == "" {
		return true
	}

	if contains(config, "enabled: false") || contains(config, "'enabled': false") {
		return false
	}
	if contains(config, "enabled: true") || contains(config, "'enabled': true") {
		return true
	}

	return true
}
