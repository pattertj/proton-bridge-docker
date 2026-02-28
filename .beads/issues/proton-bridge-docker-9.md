---
type: task
title: "Go server skeleton with embedded static files"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: []
created: 2026-02-27
---

# Go server skeleton with embedded static files

Create `webui/main.go` and `webui/go.mod`:

- HTTP server on configurable port (flag `-port`)
- Use `embed.FS` to serve `frontend/` directory at `/`
- Router with `/api/*` prefix for API routes
- Graceful shutdown on SIGTERM/SIGINT
- Initialize `go.mod` with appropriate module path
