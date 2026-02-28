---
type: task
title: "SEC-MEDIUM: Raw CLI output leaked in API responses"
status: open
priority: medium
labels: [security]
created: 2026-02-27
---

# Raw CLI output leaked in API responses

**Severity:** Medium

**Location:** `webui/handlers/bridge.go:241`, `bridge.go:271`

**Description:** When login results are ambiguous, the raw bridge CLI output is returned to the frontend: `"Output: " + truncate(combined, 300)`. This may contain internal paths, version info, error details, or partial credential data.

**Impact:** Information disclosure — internal bridge state, file paths, and potentially partial credential info returned to the browser.

**Recommendation:** Return a generic message and log the detail server-side only:

```go
log.Printf("[bridge] Ambiguous login result: %s", truncate(combined, 300))
return &LoginResult{Message: "Login submitted. Please check status."}, nil
```
