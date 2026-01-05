package frameworks_test

import (
	"os"
	"testing"
)

func TestYourKitProfilerLicense(t *testing.T) {
	licenseKey := "test-yourkit-license-key"
	os.Setenv("YOURKIT_LICENSE_KEY", licenseKey)
	defer os.Unsetenv("YOURKIT_LICENSE_KEY")

	if os.Getenv("YOURKIT_LICENSE_KEY") != licenseKey {
		t.Errorf("Expected YOURKIT_LICENSE_KEY to be %s", licenseKey)
	}
}

func TestYourKitProfilerPort(t *testing.T) {
	port := "10001"
	os.Setenv("YOURKIT_PORT", port)
	defer os.Unsetenv("YOURKIT_PORT")

	if os.Getenv("YOURKIT_PORT") != port {
		t.Errorf("Expected YOURKIT_PORT to be %s", port)
	}
}
