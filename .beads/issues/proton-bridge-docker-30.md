---
type: task
title: "SEC-HIGH: Web UI served over HTTP (no TLS)"
status: done
priority: high
labels: [security]
created: 2026-02-27
---

# Web UI served over HTTP (no TLS)

**Severity:** High

**Location:** `webui/main.go:53-58`, `docker/entrypoint.sh:34`

**Description:** The web UI server uses plain HTTP. Proton account credentials (username + password) are transmitted from the browser to the Go server in cleartext.

**Impact:** Any network observer between the user's browser and the container (e.g., on the same LAN, or via a compromised switch) can intercept Proton credentials during login.

**Recommendation:** Support optional TLS with self-signed certs via env vars. Add a prominent warning in the UI for HTTP mode:

```go
if certFile != "" && keyFile != "" {
    server.ListenAndServeTLS(certFile, keyFile)
}
```
