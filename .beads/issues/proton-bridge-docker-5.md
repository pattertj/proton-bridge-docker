---
type: task
title: "Create fallback setup.sh"
status: done
priority: medium
epic: proton-bridge-docker-1
blocked_by: [proton-bridge-docker-2]
created: 2026-02-27
---

# Create fallback setup.sh

Create `docker/setup.sh` for CLI-based account setup via `docker exec`:

- Kill running bridge process to free lock
- Launch `bridge --cli` in interactive mode
- Print instructions about restarting container after setup
- Usage: `docker exec -it protonmail-bridge /app/setup.sh`
