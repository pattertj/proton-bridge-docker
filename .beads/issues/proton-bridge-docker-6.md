---
type: task
title: "Create docker-compose.yml"
status: done
priority: medium
epic: proton-bridge-docker-1
blocked_by: [proton-bridge-docker-2]
created: 2026-02-27
---

# Create docker-compose.yml

Create example `docker-compose.yml`:

- Image: `pattertj/protonmail-bridge:latest`
- Ports: 1025:25 (SMTP), 1143:143 (IMAP), 3080:3000 (WebUI)
- Volume: `./data:/root`
- Env: `WEBUI_PORT=3000`
- Restart: unless-stopped
