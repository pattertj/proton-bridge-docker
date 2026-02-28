---
type: task
title: "REST API handlers"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-10]
created: 2026-02-27
---

# REST API handlers

Create `webui/handlers/api.go` with these endpoints:

- `GET  /api/accounts` — list configured accounts with IMAP/SMTP credentials
- `POST /api/login` — initiate login flow (body: `{username, password}`)
- `POST /api/login/2fa` — submit 2FA code (body: `{code}`)
- `POST /api/accounts/:id/mode` — toggle combined/split address mode
- `POST /api/logout/:id` — remove account

All endpoints return JSON. Login endpoints manage the RUNNING↔SETUP_MODE state transition.
