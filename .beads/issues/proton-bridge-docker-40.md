---
type: task
title: "SEC-LOW: pkill pattern too broad"
status: open
priority: low
labels: [security]
created: 2026-02-27
---

# pkill -f "bridge" pattern too broad

**Severity:** Low

**Location:** `webui/handlers/bridge.go:501`, `docker/setup.sh:8`

**Description:** `pkill -f "bridge"` will kill any process with "bridge" in its command line, including potentially the webui-server process itself (which imports the `bridge` package) or other unrelated processes.

**Impact:** Could kill unintended processes, causing cascading failures or the web UI to crash during setup mode transitions.

**Recommendation:** Use a more specific pattern:

```go
exec.Command("pkill", "-f", "/usr/bin/bridge")
```

And in `setup.sh`:
```bash
pkill -f "/usr/bin/bridge" || true
```
