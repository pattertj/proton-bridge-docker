---
type: task
title: "SEC-MEDIUM: No rate limiting on login endpoint"
status: done
priority: high
labels: [security]
created: 2026-02-27
---

# No rate limiting on login endpoint

**Severity:** Medium

**Location:** `webui/handlers/api.go:80-106`

**Description:** The `/api/login` endpoint has no rate limiting. An attacker can brute-force Proton credentials through the web UI.

**Impact:** While Proton's servers will eventually rate-limit or lock the account, the bridge CLI proxy adds a layer of obfuscation. The attacker gets feedback on whether credentials are valid.

**Recommendation:** Add a simple in-memory rate limiter (e.g., max 5 attempts per minute):

```go
var loginLimiter = rate.NewLimiter(rate.Every(12*time.Second), 5)

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
    if !loginLimiter.Allow() {
        writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts"})
        return
    }
    // ...
}
```
