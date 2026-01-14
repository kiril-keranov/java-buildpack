package frameworks

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/java-buildpack/src/java/common"
	"github.com/cloudfoundry/libbuildpack"
)

func TestDetectEnabledWithRealManifest(t *testing.T) {
	if err := os.Setenv("CF_METRICS_EXPORTER_ENABLED", "true"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	// needed to match the cf-metrics-exporter dependency in the manifest
	if err := os.Setenv("CF_STACK", "cflinuxfs4"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	manifestDir := filepath.Join("../../../")
	logger := libbuildpack.NewLogger(os.Stdout)
	manifest, err := libbuildpack.NewManifest(manifestDir, logger, time.Now())
	if err != nil {
		t.Fatalf("Failed to load manifest.yml: %v", err)
	}
	ctx := &common.Context{Manifest: manifest}
	f := NewCfMetricsExporterFramework(ctx)
	name, err := f.Detect()
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}
	if name == "" {
		t.Error("Detect() should return non-empty name when enabled")
	}
	if err := os.Unsetenv("CF_STACK"); err != nil {
		t.Fatalf("Unsetenv failed: %v", err)
	}
	if err := os.Unsetenv("CF_METRICS_EXPORTER_ENABLED"); err != nil {
		t.Fatalf("Unsetenv failed: %v", err)
	}
}

func TestDetectDisabledWithRealManifest(t *testing.T) {
	if err := os.Setenv("CF_METRICS_EXPORTER_ENABLED", "false"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	manifestDir := filepath.Join("../../../")
	logger := libbuildpack.NewLogger(os.Stdout)
	manifest, err := libbuildpack.NewManifest(manifestDir, logger, time.Now())
	if err != nil {
		t.Fatalf("Failed to load manifest.yml: %v", err)
	}
	ctx := &common.Context{Manifest: manifest}
	f := NewCfMetricsExporterFramework(ctx)
	name, err := f.Detect()
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}
	if name != "" {
		t.Error("Detect() should return empty name when disabled")
	}
	if err := os.Unsetenv("CF_METRICS_EXPORTER_ENABLED"); err != nil {
		t.Fatalf("Unsetenv failed: %v", err)
	}
}
