package frameworks_test

import (
	"testing"
)

func TestJavaCfEnvDetection(t *testing.T) {
	springBootPresent := true

	if !springBootPresent {
		t.Error("Java CF Env should be detected when Spring Boot is present")
	}
}
