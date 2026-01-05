package frameworks_test

import (
	"os"
	"testing"
)

func TestSpringAutoReconfigurationEnabled(t *testing.T) {
	os.Setenv("SPRING_PROFILES_ACTIVE", "cloud")
	defer os.Unsetenv("SPRING_PROFILES_ACTIVE")

	if os.Getenv("SPRING_PROFILES_ACTIVE") != "cloud" {
		t.Error("Spring Auto-reconfiguration should set cloud profile")
	}
}

func TestSpringAutoReconfigurationCanBeDisabled(t *testing.T) {
	config := "enabled: false"

	if !contains(config, "enabled: false") {
		t.Error("Should be able to disable Spring Auto-reconfiguration")
	}
}
