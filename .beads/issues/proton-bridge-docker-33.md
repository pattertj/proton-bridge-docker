---
type: task
title: "SEC-MEDIUM: No request body size limit"
status: done
priority: medium
labels: [security]
created: 2026-02-27
---

# No request body size limit

**Severity:** Medium

**Location:** `webui/handlers/api.go:82`, `api.go:115`

**Description:** `json.NewDecoder(r.Body).Decode(&req)` reads the full request body without any size limit. An attacker can send arbitrarily large payloads.

**Impact:** Memory exhaustion / denial of service against the web UI server.

**Recommendation:** Wrap the body with `http.MaxBytesReader`:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1024*10) // 10KB max
```
