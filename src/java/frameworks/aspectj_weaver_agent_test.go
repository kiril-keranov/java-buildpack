package frameworks_test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAspectJWeaverDetection tests AspectJ Weaver JAR detection
func TestAspectJWeaverDetection(t *testing.T) {
	// Create temporary build directory
	tmpDir, err := os.MkdirTemp("", "aspectj-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test WEB-INF/lib location
	webInfLib := filepath.Join(tmpDir, "WEB-INF", "lib")
	if err := os.MkdirAll(webInfLib, 0755); err != nil {
		t.Fatalf("Failed to create WEB-INF/lib: %v", err)
	}

	aspectjJar := filepath.Join(webInfLib, "aspectjweaver-1.9.7.jar")
	if err := os.WriteFile(aspectjJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create AspectJ JAR: %v", err)
	}

	// Verify JAR was created
	if _, err := os.Stat(aspectjJar); os.IsNotExist(err) {
		t.Error("AspectJ Weaver JAR was not created")
	}
}

// TestAspectJWeaverAopXmlDetection tests aop.xml configuration detection
func TestAspectJWeaverAopXmlDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aspectj-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test META-INF/aop.xml location
	metaInf := filepath.Join(tmpDir, "META-INF")
	if err := os.MkdirAll(metaInf, 0755); err != nil {
		t.Fatalf("Failed to create META-INF: %v", err)
	}

	aopXml := filepath.Join(metaInf, "aop.xml")
	aopContent := `<?xml version="1.0" encoding="UTF-8"?>
<aspectj>
    <aspects>
        <aspect name="com.example.MyAspect"/>
    </aspects>
</aspectj>`

	if err := os.WriteFile(aopXml, []byte(aopContent), 0644); err != nil {
		t.Fatalf("Failed to create aop.xml: %v", err)
	}

	// Verify aop.xml was created
	if _, err := os.Stat(aopXml); os.IsNotExist(err) {
		t.Error("aop.xml configuration was not created")
	}
}

// TestAspectJWeaverWebInfAopXmlDetection tests aop.xml in WEB-INF/classes
func TestAspectJWeaverWebInfAopXmlDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aspectj-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test WEB-INF/classes/META-INF/aop.xml location
	webInfMetaInf := filepath.Join(tmpDir, "WEB-INF", "classes", "META-INF")
	if err := os.MkdirAll(webInfMetaInf, 0755); err != nil {
		t.Fatalf("Failed to create WEB-INF/classes/META-INF: %v", err)
	}

	aopXml := filepath.Join(webInfMetaInf, "aop.xml")
	aopContent := `<?xml version="1.0" encoding="UTF-8"?>
<aspectj>
    <weaver options="-verbose -showWeaveInfo">
        <include within="com.example..*"/>
    </weaver>
</aspectj>`

	if err := os.WriteFile(aopXml, []byte(aopContent), 0644); err != nil {
		t.Fatalf("Failed to create WEB-INF aop.xml: %v", err)
	}

	// Verify aop.xml was created in WEB-INF location
	if _, err := os.Stat(aopXml); os.IsNotExist(err) {
		t.Error("WEB-INF aop.xml configuration was not created")
	}
}

// TestAspectJWeaverJarSearch tests searching multiple locations for AspectJ JAR
func TestAspectJWeaverJarSearch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aspectj-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple search locations
	locations := []string{
		filepath.Join(tmpDir, "WEB-INF", "lib"),
		filepath.Join(tmpDir, "lib"),
		filepath.Join(tmpDir, "BOOT-INF", "lib"),
	}

	for _, loc := range locations {
		if err := os.MkdirAll(loc, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", loc, err)
		}
	}

	// Place AspectJ JAR in lib directory
	libDir := filepath.Join(tmpDir, "lib")
	aspectjJar := filepath.Join(libDir, "aspectjweaver-1.9.7.jar")
	if err := os.WriteFile(aspectjJar, []byte("mock jar"), 0644); err != nil {
		t.Fatalf("Failed to create AspectJ JAR: %v", err)
	}

	// Verify JAR exists
	if _, err := os.Stat(aspectjJar); os.IsNotExist(err) {
		t.Error("AspectJ Weaver JAR not found in lib directory")
	}
}

// TestAspectJWeaverJarNaming tests different AspectJ JAR naming patterns
func TestAspectJWeaverJarNaming(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aspectj-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	libDir := filepath.Join(tmpDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("Failed to create lib directory: %v", err)
	}

	// Test various valid naming patterns
	validNames := []string{
		"aspectjweaver-1.9.7.jar",
		"aspectjweaver-1.9.7.RELEASE.jar",
		"aspectjweaver-1.9.8.M1.jar",
	}

	for _, name := range validNames {
		jarPath := filepath.Join(libDir, name)
		if err := os.WriteFile(jarPath, []byte("mock jar"), 0644); err != nil {
			t.Errorf("Failed to create JAR %s: %v", name, err)
			continue
		}

		// Verify JAR was created
		if _, err := os.Stat(jarPath); os.IsNotExist(err) {
			t.Errorf("JAR %s was not created", name)
		}

		// Clean up for next iteration
		os.Remove(jarPath)
	}
}
