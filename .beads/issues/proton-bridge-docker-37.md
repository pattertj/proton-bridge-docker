---
type: task
title: "SEC-LOW: State file world-readable (0644)"
status: open
priority: low
labels: [security]
created: 2026-02-27
---

# State file world-readable (0644)

**Severity:** Low

**Location:** `webui/handlers/state.go:50` — `os.WriteFile(stateFilePath, data, 0644)`

**Description:** The state file is written with mode `0644` (world-readable). It contains account email addresses.

**Impact:** Other processes in the container (or anyone with volume access) can read which Proton accounts are configured.

**Recommendation:** Use `0600`:

```go
os.WriteFile(stateFilePath, data, 0600)
```
