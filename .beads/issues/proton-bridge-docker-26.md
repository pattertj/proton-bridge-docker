---
type: task
title: "SEC-CRITICAL: Add authentication to Web UI"
status: done
priority: critical
labels: [security]
created: 2026-02-27
---

# Add authentication to Web UI

**Severity:** Critical

**Location:** `webui/main.go:37-44`, all API endpoints

**Description:** Every API endpoint (`/api/login`, `/api/accounts`, `/api/status`, `/api/logout/{id}`) is completely unauthenticated. Anyone with network access to port 3000/3080 can log in with Proton credentials, view bridge-generated passwords, remove accounts, or trigger state changes.

**Impact:** On a shared network or misconfigured Unraid setup, any device can take over the bridge — view email credentials, remove accounts, or initiate login with attacker-controlled credentials. The Unraid template exposes port 3080 on `0.0.0.0`.

**Recommendation:** Add a shared secret / basic auth token configured via environment variable. Better: generate a random admin token on first run, display it in container logs, and require it as a `Bearer` token or via a login cookie.

```go
func authMiddleware(next http.Handler) http.Handler {
    token := os.Getenv("WEBUI_AUTH_TOKEN")
    if token == "" { return next }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer "+token {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```
