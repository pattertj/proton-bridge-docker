package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestNoRawCLIOutputInResponses(t *testing.T) {
	// Verify no API responses contain raw CLI output
	data, err := os.ReadFile("bridge.go")
	if err != nil {
		t.Fatalf("failed to read bridge.go: %v", err)
	}
	src := string(data)

	// LoginResult should never contain raw output
	dangerousPatterns := []string{
		`"Output: "`,
		`"output: "`,
	}
	for _, pat := range dangerousPatterns {
		if strings.Contains(src, pat) {
			t.Errorf("found raw CLI output in API response pattern: %q", pat)
		}
	}
}

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
