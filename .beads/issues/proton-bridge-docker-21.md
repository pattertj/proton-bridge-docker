---
type: task
title: "Configure Docker Hub + GHCR + secrets"
status: done
priority: medium
epic: proton-bridge-docker-17
blocked_by: [proton-bridge-docker-18]
created: 2026-02-27
---

# Configure Docker Hub + GHCR + secrets

Manual setup steps:

1. Create Docker Hub repo `pattertj/protonmail-bridge`
2. Generate Docker Hub access token
3. Add GitHub secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`
4. GHCR uses `GITHUB_TOKEN` automatically
5. Test: push to main, verify image builds and pushes
