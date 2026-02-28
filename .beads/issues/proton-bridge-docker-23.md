---
type: task
title: "Create Unraid CA template XML"
status: done
priority: low
epic: proton-bridge-docker-22
blocked_by: []
created: 2026-02-27
---

# Create Unraid CA template XML

Create `unraid/protonmail-bridge.xml`:

- Container config: ports (SMTP 1025:25, IMAP 1143:143, WebUI 3080:3000)
- Volume: `/mnt/user/appdata/protonmail-bridge` → `/root`
- WebUI: `http://[IP]:[PORT:3000]`
- Overview text explaining Bridge, requirements (paid plan), use cases
- Branches: latest (Debian), alpine (future)
