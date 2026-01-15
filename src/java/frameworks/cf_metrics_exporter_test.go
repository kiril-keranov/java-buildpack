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

func TestSupplyPlacesJarCorrectly(t *testing.T) {
	if err := os.Setenv("CF_METRICS_EXPORTER_ENABLED", "true"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	if err := os.Setenv("CF_STACK", "cflinuxfs4"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	manifestDir := filepath.Join("../../../")
	logger := libbuildpack.NewLogger(os.Stdout)
	manifest, err := libbuildpack.NewManifest(manifestDir, logger, time.Now())
	if err != nil {
		t.Fatalf("Failed to load manifest.yml: %v", err)
	}
	tmpDepDir, err := os.MkdirTemp("", "cf_metrics_exporter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dep dir: %v", err)
	}
	defer os.RemoveAll(tmpDepDir)
	args := []string{"", "", tmpDepDir, "0"}
	ctx := &common.Context{Manifest: manifest}
	ctx.Stager = libbuildpack.NewStager(args, logger, manifest)
	// Do not set ctx.Installer, pass mock to framework constructor
	f := &CfMetricsExporterFramework{ctx: ctx, installer: &mockInstallerJar{}}
	if err := f.Supply(); err != nil {
		t.Fatalf("Supply() error: %v", err)
	}
	// Check the JAR file exists directly in cf_metrics_exporter
	jarName := "cf-metrics-exporter-0.7.1.jar" // adjust if version changes in manifest
	jarPath := filepath.Join(tmpDepDir, "cf_metrics_exporter", jarName)
	// Print directory contents for debugging
	dirPath := filepath.Join(tmpDepDir, "cf_metrics_exporter")
	dirEntries, dirErr := os.ReadDir(dirPath)
	if dirErr != nil {
		t.Errorf("Error reading cf_metrics_exporter dir: %v", dirErr)
	} else {
		for _, entry := range dirEntries {
			t.Logf("Found in cf_metrics_exporter: %s", entry.Name())
		}
	}
	if fi, err := os.Stat(jarPath); err != nil {
		t.Errorf("JAR file not found at expected path: %s, error: %v", jarPath, err)
	} else if fi.IsDir() {
		t.Errorf("Expected file but found directory at: %s", jarPath)
	}
	// Check there is NOT a directory named after the JAR inside cf_metrics_exporter
	badDir := filepath.Join(tmpDepDir, "cf_metrics_exporter", jarName)
	if fi, err := os.Stat(badDir); err == nil && fi.IsDir() {
		t.Errorf("Unexpected directory found: %s", badDir)
	}
	if err := os.Unsetenv("CF_STACK"); err != nil {
		t.Fatalf("Unsetenv failed: %v", err)
	}
	if err := os.Unsetenv("CF_METRICS_EXPORTER_ENABLED"); err != nil {
		t.Fatalf("Unsetenv failed: %v", err)
	}
}

type mockInstallerJar struct{}

func (m *mockInstallerJar) InstallDependency(dep libbuildpack.Dependency, outputDir string) error {
	// Simulate a successful download: create the agent dir and a JAR file with the expected name
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	jarName := "cf-metrics-exporter-0.7.1.jar"
	jarPath := filepath.Join(outputDir, jarName)
	f, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	return f.Close()
}
