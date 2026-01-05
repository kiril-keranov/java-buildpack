package frameworks_test

import (
	"testing"
)

func TestTakipiAgentCredentials(t *testing.T) {
	credentials := map[string]interface{}{
		"secret_key": "test-secret-key-xyz",
	}

	if key, ok := credentials["secret_key"].(string); !ok || key == "" {
		t.Error("secret_key is required for Takipi")
	}
}
