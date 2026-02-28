---
type: task
title: "Create entrypoint.sh"
status: done
priority: high
epic: proton-bridge-docker-1
blocked_by: [proton-bridge-docker-2]
created: 2026-02-27
---

# Create entrypoint.sh

Create `docker/entrypoint.sh` that handles in order:

1. GPG keyring init (skip if `/root/.gnupg` exists with keys)
2. `pass init` with the GPG key ID (skip if `/root/.password-store` exists)
3. Read env vars: `CONTAINER_SMTP_PORT` (default 25), `CONTAINER_IMAP_PORT` (default 143), `WEBUI_PORT` (default 3000)
4. Start socat proxies: forward `0.0.0.0:SMTP_PORT` → `127.0.0.1:1025`, `0.0.0.0:IMAP_PORT` → `127.0.0.1:1143`
5. Start webui-server (Phase 2, skip for now)
6. `exec /usr/bin/bridge --noninteractive`
