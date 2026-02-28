package handlers

import (
	"os"
	"strings"
	"testing"
)

func TestStateFilePermissions(t *testing.T) {
	// Verify the source code uses 0600 not 0644
	data, err := os.ReadFile("state.go")
	if err != nil {
		t.Fatalf("failed to read state.go: %v", err)
	}
	src := string(data)

	if strings.Contains(src, "0644") {
		t.Error("state.go still uses 0644 permissions; should be 0600")
	}
	if !strings.Contains(src, "0600") {
		t.Error("state.go should use 0600 for state file permissions")
	}
}
