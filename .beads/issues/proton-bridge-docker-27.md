---
type: task
title: "SEC-HIGH: Sanitize CLI command inputs to prevent injection"
status: done
priority: critical
labels: [security]
created: 2026-02-27
---

# Sanitize CLI command inputs to prevent injection

**Severity:** High

**Location:** `webui/handlers/bridge.go:310`, `bridge.go:331`

**Description:** `ToggleMode` and `LogoutAccount` concatenate user-supplied `accountID` directly into CLI commands: `"change mode " + accountID` and `"logout " + accountID`. Since the bridge CLI is a REPL, an attacker could inject newlines or additional commands. For example, `accountID = "0\ndelete 0"` would send two commands.

**Impact:** Arbitrary bridge CLI command execution — delete accounts, change settings, or potentially exfiltrate data via crafted CLI sequences.

**Recommendation:** Validate `accountID` to contain only alphanumeric characters/hyphens, and strip newlines:

```go
func sanitizeID(id string) (string, error) {
    if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-]+$`, id); !matched {
        return "", fmt.Errorf("invalid account ID")
    }
    return id, nil
}
```
