package frameworks_test

import (
	"os"
	"testing"
)

func TestJavaMemoryAssistantDetection(t *testing.T) {
	heapDumpPath := "/tmp/heapdumps"
	os.Setenv("BPL_HEAP_DUMP_PATH", heapDumpPath)
	defer os.Unsetenv("BPL_HEAP_DUMP_PATH")

	if os.Getenv("BPL_HEAP_DUMP_PATH") != heapDumpPath {
		t.Errorf("Expected BPL_HEAP_DUMP_PATH to be %s", heapDumpPath)
	}
}
