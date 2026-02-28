package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestNoCredentialLogging(t *testing.T) {
	// Read the bridge.go source to verify no credential-adjacent logging
	data, err := os.ReadFile("bridge.go")
	if err != nil {
		t.Fatalf("failed to read bridge.go: %v", err)
	}
	src := string(data)

	// After sending password, we should not log the output/response content
	dangerousPatterns := []string{
		`Password response:`,
		`Password output:`,
		`Username response:`,
	}

	for _, pat := range dangerousPatterns {
		re := regexp.MustCompile(pat)
		if re.MatchString(src) {
			t.Errorf("found credential-adjacent logging pattern %q in bridge.go", pat)
		}
	}

	// Verify we DO have safe logging instead
	safePatterns := []string{
		"Password submitted",
		"Username submitted",
	}
	for _, pat := range safePatterns {
		if !strings.Contains(src, pat) {
			t.Errorf("expected safe log message %q not found in bridge.go", pat)
		}
	}
}
