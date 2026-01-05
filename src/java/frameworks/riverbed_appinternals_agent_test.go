package frameworks_test

import (
	"testing"
)

func TestRiverbedAppInternalsCredentials(t *testing.T) {
	credentials := map[string]interface{}{
		"analysis_server": "riverbed.example.com",
	}

	if server, ok := credentials["analysis_server"].(string); !ok || server == "" {
		t.Error("analysis_server is required for Riverbed AppInternals")
	}
}
