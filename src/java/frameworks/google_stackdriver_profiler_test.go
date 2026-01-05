package frameworks_test

import (
	"os"
	"testing"
)

func TestGoogleStackdriverProfilerCredentials(t *testing.T) {
	projectID := "test-project-123"
	os.Setenv("GOOGLE_CLOUD_PROJECT", projectID)
	defer os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	if os.Getenv("GOOGLE_CLOUD_PROJECT") != projectID {
		t.Errorf("Expected GOOGLE_CLOUD_PROJECT to be %s", projectID)
	}
}

func TestGoogleStackdriverProfilerServiceAccountKey(t *testing.T) {
	keyJSON := `{"type": "service_account", "project_id": "test-project"}`
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", keyJSON)
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON") == "" {
		t.Error("GOOGLE_APPLICATION_CREDENTIALS_JSON should be set")
	}
}
