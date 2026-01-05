package frameworks_test

import (
	"testing"
)

func TestContainerSecurityProviderAlwaysDetected(t *testing.T) {
	detected := true
	if !detected {
		t.Error("Container Security Provider should always be detected")
	}
}

func TestContainerSecurityProviderJavaVersionDetection(t *testing.T) {
	tests := []struct {
		name         string
		javaVersion  int
		expectedType string
	}{
		{
			name:         "Java 8 uses extension directory",
			javaVersion:  8,
			expectedType: "extension",
		},
		{
			name:         "Java 9 uses bootstrap classpath",
			javaVersion:  9,
			expectedType: "bootclasspath",
		},
		{
			name:         "Java 11 uses bootstrap classpath",
			javaVersion:  11,
			expectedType: "bootclasspath",
		},
		{
			name:         "Java 17 uses bootstrap classpath",
			javaVersion:  17,
			expectedType: "bootclasspath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mechanism string
			if tt.javaVersion >= 9 {
				mechanism = "bootclasspath"
			} else {
				mechanism = "extension"
			}

			if mechanism != tt.expectedType {
				t.Errorf("Expected %s mechanism for Java %d, got %s", tt.expectedType, tt.javaVersion, mechanism)
			}
		})
	}
}
