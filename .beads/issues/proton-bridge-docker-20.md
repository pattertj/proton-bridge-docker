---
type: task
title: "Release workflow"
status: done
priority: medium
epic: proton-bridge-docker-17
blocked_by: []
created: 2026-02-27
---

# Release workflow

Create `.github/workflows/release.yml`:

- Trigger: push of `v*` tags
- Extract bridge version from Dockerfile
- Create GitHub Release with docker pull commands and upstream release notes link
- Auto-generate release notes from commits
