---
type: task
title: "SEC-LOW: No security response headers"
status: done
priority: low
labels: [security]
created: 2026-02-27
---

# No security response headers

**Severity:** Low

**Location:** `webui/handlers/api.go` — all endpoints

**Description:** No security headers (CORS, X-Content-Type-Options, X-Frame-Options, CSP) are configured. The default Go behavior is secure (blocks cross-origin), but there's no defense-in-depth or explicit policy.

**Impact:** Low risk currently, but missing headers leave the UI vulnerable to clickjacking if embedded in an iframe, and provide no Content Security Policy.

**Recommendation:** Add a middleware that sets security headers:

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "SAMEORIGIN")
        w.Header().Set("Referrer-Policy", "no-referrer")
        next.ServeHTTP(w, r)
    })
}
```
