---
type: task
title: "State persistence (webui-state.json)"
status: done
priority: medium
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-9]
created: 2026-02-27
---

# State persistence (webui-state.json)

Handle container restarts gracefully:

## On startup
1. Check if `/root/.config/protonmail/bridge/` exists
2. If accounts configured → start bridge non-interactive, serve dashboard
3. If no accounts → serve setup wizard

## State file: `/root/.webui-state.json`
```json
{
  "setup_complete": true,
  "last_bridge_version": "3.22.0",
  "accounts": ["user@proton.me"]
}
```

Read on startup, update after login/logout operations.
