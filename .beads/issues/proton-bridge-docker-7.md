---
type: task
title: "Test Docker foundation end-to-end"
status: done
priority: high
epic: proton-bridge-docker-1
blocked_by: [proton-bridge-docker-2, proton-bridge-docker-3, proton-bridge-docker-4, proton-bridge-docker-5, proton-bridge-docker-6]
created: 2026-02-27
---

# Test Docker foundation end-to-end

Verify Phase 1 works:

1. `docker compose up` builds and starts successfully
2. GPG key and pass store initialize on first run
3. socat proxies are running and forwarding ports
4. `docker exec -it protonmail-bridge /app/setup.sh` launches CLI
5. After login via CLI, IMAP/SMTP are reachable from host
6. Container restart preserves config (volume persistence)
