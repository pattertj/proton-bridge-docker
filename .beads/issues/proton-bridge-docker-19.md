---
type: task
title: "Upstream version check workflow"
status: done
priority: medium
epic: proton-bridge-docker-17
blocked_by: []
created: 2026-02-27
---

# Upstream version check workflow

Create `.github/workflows/check-upstream.yml`:

- Cron: every Monday 9am UTC + manual trigger
- Compare `BRIDGE_VERSION` in Dockerfile vs latest GitHub release of ProtonMail/proton-bridge
- If newer version found: update Dockerfile, create PR with release notes link
- Branch naming: `auto/bridge-vX.Y.Z`
