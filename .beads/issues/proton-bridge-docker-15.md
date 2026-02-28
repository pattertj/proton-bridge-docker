---
type: task
title: "Update Dockerfile for webui build stage"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-9, proton-bridge-docker-13]
created: 2026-02-27
---

# Update Dockerfile for webui build stage

Add Stage 2 to `docker/Dockerfile`:

- `webui-builder` stage: `golang:1.24-bookworm`, copy `webui/`, build with `CGO_ENABLED=0`
- Copy `/webui-server` binary to runtime stage
- Update entrypoint to launch webui-server before bridge
