---
type: task
title: "Create multi-stage Dockerfile"
status: done
priority: high
epic: proton-bridge-docker-1
blocked_by: []
created: 2026-02-27
---

# Create multi-stage Dockerfile

Create `docker/Dockerfile` with 3 stages:

1. **bridge-builder** — `golang:1.24-bookworm`, clone proton-bridge at `BRIDGE_VERSION` arg (default v3.22.0), `make build-nogui`
2. **webui-builder** — `golang:1.24-bookworm`, build webui Go binary (placeholder for now, skip until Phase 2)
3. **runtime** — `debian:bookworm-slim`, install `pass gnupg socat ca-certificates procps`, copy binaries, expose ports 25/143/3000, volume `/root`, OCI labels

For Phase 1, omit the webui-builder stage and just build bridge + runtime.
