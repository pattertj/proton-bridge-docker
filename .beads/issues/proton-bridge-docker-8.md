---
type: epic
title: "Phase 2: Web UI"
status: done
priority: high
blocked_by: [proton-bridge-docker-1]
created: 2026-02-27
---

# Phase 2: Web UI

Build a Go-based web UI that automates bridge CLI interaction, eliminating the need for `docker exec`. Includes REST API, Bridge CLI process management, and a React SPA frontend.

## Acceptance Criteria
- Go HTTP server with embedded static frontend
- Bridge CLI interaction layer manages subprocess I/O (login, 2FA, account info)
- State machine: RUNNING ↔ SETUP_MODE for single-instance constraint
- React SPA: setup wizard, sync progress, dashboard with credentials
- Full login flow works through web UI (including 2FA)
- Container restart loads dashboard with saved account

## Subtasks
- proton-bridge-docker-9: Go server skeleton + embedded static files
- proton-bridge-docker-10: Bridge CLI interaction layer
- proton-bridge-docker-11: REST API handlers
- proton-bridge-docker-12: Status/health endpoints
- proton-bridge-docker-13: Frontend SPA (setup wizard + dashboard)
- proton-bridge-docker-14: State persistence (webui-state.json)
- proton-bridge-docker-15: Update Dockerfile for webui build stage
- proton-bridge-docker-16: Test Web UI end-to-end
