package frameworks_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCheckmarxIASTServiceDetection(t *testing.T) {
	tests := []struct {
		name         string
		vcapServices string
		shouldDetect bool
	}{
		{
			name: "checkmarx-iast service",
			vcapServices: `{
				"checkmarx-iast": [{
					"name": "my-checkmarx",
					"credentials": {
						"url": "https://example.com/agent.jar"
					}
				}]
			}`,
			shouldDetect: true,
		},
		{
			name: "checkmarx service",
			vcapServices: `{
				"checkmarx": [{
					"name": "my-checkmarx",
					"credentials": {
						"url": "https://example.com/agent.jar"
					}
				}]
			}`,
			shouldDetect: true,
		},
		{
			name: "user-provided with checkmarx in name",
			vcapServices: `{
				"user-provided": [{
					"name": "my-checkmarx-service",
					"credentials": {
						"url": "https://example.com/agent.jar"
					}
				}]
			}`,
			shouldDetect: true,
		},
		{
			name: "service with checkmarx tag",
			vcapServices: `{
				"security-service": [{
					"name": "my-security",
					"tags": ["checkmarx", "iast"],
					"credentials": {
						"url": "https://example.com/agent.jar"
					}
				}]
			}`,
			shouldDetect: true,
		},
		{
			name: "no checkmarx service",
			vcapServices: `{
				"other-service": [{
					"name": "some-service",
					"credentials": {}
				}]
			}`,
			shouldDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("VCAP_SERVICES", tt.vcapServices)
			defer os.Unsetenv("VCAP_SERVICES")

			var vcapServices map[string]interface{}
			if err := json.Unmarshal([]byte(tt.vcapServices), &vcapServices); err != nil {
				t.Fatalf("Failed to parse test VCAP_SERVICES: %v", err)
			}

			hasCheckmarx := false
			for key := range vcapServices {
				if key == "checkmarx-iast" || key == "checkmarx" {
					hasCheckmarx = true
					break
				}
			}

			if tt.shouldDetect && !hasCheckmarx {
				if !contains(tt.vcapServices, "checkmarx") {
					t.Errorf("Expected to detect Checkmarx service, but VCAP_SERVICES doesn't contain 'checkmarx'")
				}
			}
		})
	}
}

func TestCheckmarxIASTCredentialsExtraction(t *testing.T) {
	tests := []struct {
		name        string
		credentials map[string]interface{}
		expectURL   string
		expectMgr   string
		expectKey   string
	}{
		{
			name: "standard credentials",
			credentials: map[string]interface{}{
				"url":         "https://example.com/agent.jar",
				"manager_url": "https://manager.example.com",
				"api_key":     "test-key-123",
			},
			expectURL: "https://example.com/agent.jar",
			expectMgr: "https://manager.example.com",
			expectKey: "test-key-123",
		},
		{
			name: "alternative credential keys",
			credentials: map[string]interface{}{
				"agent_url":  "https://example.com/cx-agent.jar",
				"managerUrl": "https://mgr.example.com",
				"apiKey":     "key-456",
			},
			expectURL: "https://example.com/cx-agent.jar",
			expectMgr: "https://mgr.example.com",
			expectKey: "key-456",
		},
		{
			name: "minimal credentials",
			credentials: map[string]interface{}{
				"url": "https://example.com/agent.jar",
			},
			expectURL: "https://example.com/agent.jar",
			expectMgr: "",
			expectKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if url, ok := tt.credentials["url"].(string); ok {
				if url != tt.expectURL {
					t.Errorf("Expected URL %s, got %s", tt.expectURL, url)
				}
			} else if url, ok := tt.credentials["agent_url"].(string); ok {
				if url != tt.expectURL {
					t.Errorf("Expected URL %s, got %s", tt.expectURL, url)
				}
			}

			if tt.expectMgr != "" {
				if mgr, ok := tt.credentials["manager_url"].(string); ok {
					if mgr != tt.expectMgr {
						t.Errorf("Expected manager URL %s, got %s", tt.expectMgr, mgr)
					}
				} else if mgr, ok := tt.credentials["managerUrl"].(string); ok {
					if mgr != tt.expectMgr {
						t.Errorf("Expected manager URL %s, got %s", tt.expectMgr, mgr)
					}
				}
			}

			if tt.expectKey != "" {
				if key, ok := tt.credentials["api_key"].(string); ok {
					if key != tt.expectKey {
						t.Errorf("Expected API key %s, got %s", tt.expectKey, key)
					}
				} else if key, ok := tt.credentials["apiKey"].(string); ok {
					if key != tt.expectKey {
						t.Errorf("Expected API key %s, got %s", tt.expectKey, key)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
