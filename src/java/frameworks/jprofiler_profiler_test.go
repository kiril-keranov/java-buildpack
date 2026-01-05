package frameworks_test

import (
	"os"
	"testing"
)

func TestJProfilerLicenseKey(t *testing.T) {
	licenseKey := "test-license-key-12345"
	os.Setenv("JPROFILER_LICENSE_KEY", licenseKey)
	defer os.Unsetenv("JPROFILER_LICENSE_KEY")

	if os.Getenv("JPROFILER_LICENSE_KEY") != licenseKey {
		t.Errorf("Expected JPROFILER_LICENSE_KEY to be %s", licenseKey)
	}
}

func TestJProfilerPort(t *testing.T) {
	port := "8849"
	os.Setenv("JPROFILER_PORT", port)
	defer os.Unsetenv("JPROFILER_PORT")

	if os.Getenv("JPROFILER_PORT") != port {
		t.Errorf("Expected JPROFILER_PORT to be %s", port)
	}
}
