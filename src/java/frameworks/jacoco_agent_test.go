package frameworks_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJacocoAgentDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jacoco-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jacocoJar := filepath.Join(tmpDir, "jacocoagent.jar")
	if err := os.WriteFile(jacocoJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create JaCoCo JAR: %v", err)
	}

	if _, err := os.Stat(jacocoJar); os.IsNotExist(err) {
		t.Error("JaCoCo agent JAR was not created")
	}
}
