package frameworks_test

import (
	"testing"
)

func TestIntroscopeAgentConfiguration(t *testing.T) {
	credentials := map[string]interface{}{
		"agent_manager_url": "introscope-em.example.com",
		"agent_name":        "MyApp",
	}

	if url, ok := credentials["agent_manager_url"].(string); !ok || url == "" {
		t.Error("agent_manager_url is required for Introscope")
	}

	if name, ok := credentials["agent_name"].(string); !ok || name == "" {
		t.Error("agent_name is required for Introscope")
	}
}
