package frameworks_test

import (
	"encoding/json"
	"testing"
)

func TestDynatraceManifestParsing(t *testing.T) {
	manifestJSON := `{
		"tenantToken": "test-token-123",
		"communicationEndpoints": [
			"https://endpoint1.dynatrace.com",
			"https://endpoint2.dynatrace.com"
		],
		"technologies": {
			"process": {
				"linux-x86-64": []
			}
		}
	}`

	var manifest map[string]interface{}
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("Failed to parse Dynatrace manifest: %v", err)
	}

	if tenantToken, ok := manifest["tenantToken"].(string); !ok || tenantToken != "test-token-123" {
		t.Error("Expected tenantToken to be 'test-token-123'")
	}

	if endpoints, ok := manifest["communicationEndpoints"].([]interface{}); !ok || len(endpoints) != 2 {
		t.Error("Expected 2 communication endpoints")
	}
}

func TestDynatraceCredentials(t *testing.T) {
	credentials := map[string]interface{}{
		"apitoken":      "dt0c01.test.token",
		"environmentid": "abc12345",
		"apiurl":        "https://abc12345.live.dynatrace.com/api",
	}

	if _, ok := credentials["apitoken"]; !ok {
		t.Error("apitoken is required for Dynatrace")
	}
	if _, ok := credentials["environmentid"]; !ok {
		t.Error("environmentid is required for Dynatrace")
	}
}
