package frameworks_test

import (
	"os"
	"testing"
)

func TestJavaOptsWriterBasicOptions(t *testing.T) {
	javaOpts := "-Xmx512M -Xms256M"
	os.Setenv("JAVA_OPTS", javaOpts)
	defer os.Unsetenv("JAVA_OPTS")

	if os.Getenv("JAVA_OPTS") != javaOpts {
		t.Errorf("Expected JAVA_OPTS to be %s", javaOpts)
	}
}
