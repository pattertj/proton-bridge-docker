---
type: task
title: "Build and push workflow"
status: done
priority: medium
epic: proton-bridge-docker-17
blocked_by: []
created: 2026-02-27
---

# Build and push workflow

Create `.github/workflows/build-and-push.yml`:

- Trigger: push to main, version tags, PRs
- Multi-arch: linux/amd64, linux/arm64 (QEMU + Buildx)
- Push to Docker Hub (`pattertj/protonmail-bridge`) and GHCR
- Tags: semver, sha, latest on default branch
- GHA cache for layer caching
- Update Docker Hub README on main push
