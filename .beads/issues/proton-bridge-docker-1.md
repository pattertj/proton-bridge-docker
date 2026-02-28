---
type: epic
title: "Phase 1: Docker Foundation"
status: done
priority: high
created: 2026-02-27
---

# Phase 1: Docker Foundation

Get Proton Mail Bridge running in a Docker container with GPG/pass keychain, socat proxies for IMAP/SMTP, and a fallback CLI setup script.

## Acceptance Criteria
- Multi-stage Dockerfile builds bridge from source (CLI-only, no Qt)
- Entrypoint handles GPG init, pass init, socat proxies, bridge launch
- IMAP (143) and SMTP (25) are accessible from other containers
- Fallback `setup.sh` allows `docker exec` based account setup
- `docker-compose.yml` example works out of the box

## Subtasks
- proton-bridge-docker-2: Create Dockerfile
- proton-bridge-docker-3: Create entrypoint.sh
- proton-bridge-docker-4: Create GPGparams.txt
- proton-bridge-docker-5: Create setup.sh
- proton-bridge-docker-6: Create docker-compose.yml
- proton-bridge-docker-7: Test Docker foundation end-to-end
