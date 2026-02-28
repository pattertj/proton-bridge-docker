---
type: task
title: "Status and health endpoints"
status: done
priority: medium
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-10]
created: 2026-02-27
---

# Status and health endpoints

Create `webui/handlers/status.go`:

- `GET /api/status` — bridge state (running/syncing/setup), account list, sync progress percentage, socat PID health
- `GET /api/health` — simple healthcheck (200 OK if webui server is up, include bridge process status)

Status endpoint should:
- Check if bridge process is alive
- Check socat proxy PIDs
- Report current state machine state
- Include sync progress if syncing
