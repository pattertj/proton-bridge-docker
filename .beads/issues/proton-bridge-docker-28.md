---
type: task
title: "SEC-HIGH: Remove credential-adjacent logging"
status: open
priority: critical
labels: [security]
created: 2026-02-27
---

# Remove credential-adjacent logging

**Severity:** High

**Location:** `webui/handlers/bridge.go:207-208`

**Description:** `log.Printf("[bridge] Password response: %s", truncate(output, 200))` logs the bridge CLI output immediately after the password is sent. While this logs the *response* (not the password itself), the bridge CLI may echo the password or include it in error messages. The username is also logged at line 201.

**Impact:** Proton account credentials potentially visible in container logs (`docker logs`), which may be persisted, forwarded to log aggregators, or visible to other users on the system.

**Recommendation:** Remove or redact password-adjacent logging:

```go
log.Printf("[bridge] Password submitted, awaiting response...")
```
