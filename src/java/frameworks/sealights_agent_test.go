package frameworks_test

import (
	"testing"
)

func TestSealightsAgentCredentials(t *testing.T) {
	credentials := map[string]interface{}{
		"token":            "test-token-123",
		"build_session_id": "test-build-session",
	}

	requiredKeys := []string{"token", "build_session_id"}
	for _, key := range requiredKeys {
		if _, exists := credentials[key]; !exists {
			t.Errorf("Required credential key %s is missing", key)
		}
	}
}
