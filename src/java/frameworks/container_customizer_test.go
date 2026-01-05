package frameworks_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainerCustomizerDetectsSpringBootWAR(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "container-customizer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	webInfDir := filepath.Join(tmpDir, "WEB-INF")
	bootInfDir := filepath.Join(tmpDir, "BOOT-INF")
	webInfLib := filepath.Join(webInfDir, "lib")
	bootInfLib := filepath.Join(bootInfDir, "lib")

	if err := os.MkdirAll(webInfLib, 0755); err != nil {
		t.Fatalf("Failed to create WEB-INF/lib: %v", err)
	}
	if err := os.MkdirAll(bootInfLib, 0755); err != nil {
		t.Fatalf("Failed to create BOOT-INF/lib: %v", err)
	}

	springBootJar := filepath.Join(webInfLib, "spring-boot-2.7.0.jar")
	if err := os.WriteFile(springBootJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create Spring Boot JAR: %v", err)
	}

	if _, err := os.Stat(webInfDir); os.IsNotExist(err) {
		t.Error("WEB-INF directory was not created")
	}
	if _, err := os.Stat(bootInfDir); os.IsNotExist(err) {
		t.Error("BOOT-INF directory was not created")
	}
	if _, err := os.Stat(springBootJar); os.IsNotExist(err) {
		t.Error("Spring Boot JAR was not created")
	}
}

func TestContainerCustomizerChecksMultipleLibLocations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "container-customizer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	locations := []string{
		filepath.Join(tmpDir, "WEB-INF", "lib"),
		filepath.Join(tmpDir, "BOOT-INF", "lib"),
	}

	for _, loc := range locations {
		if err := os.MkdirAll(loc, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", loc, err)
		}
	}

	springBootJar := filepath.Join(locations[0], "spring-boot-starter-web-2.7.0.jar")
	if err := os.WriteFile(springBootJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create Spring Boot JAR: %v", err)
	}

	if _, err := os.Stat(springBootJar); os.IsNotExist(err) {
		t.Error("Spring Boot JAR not found")
	}
}

func TestContainerCustomizerIgnoresNonSpringBootWAR(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "container-customizer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	webInfLib := filepath.Join(tmpDir, "WEB-INF", "lib")
	if err := os.MkdirAll(webInfLib, 0755); err != nil {
		t.Fatalf("Failed to create WEB-INF/lib: %v", err)
	}

	otherJar := filepath.Join(webInfLib, "servlet-api-3.1.0.jar")
	if err := os.WriteFile(otherJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create JAR: %v", err)
	}

	if _, err := os.Stat(webInfLib); os.IsNotExist(err) {
		t.Error("WEB-INF/lib directory was not created")
	}
}
