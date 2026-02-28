---
type: task
title: "Create GPGparams.txt"
status: done
priority: high
epic: proton-bridge-docker-1
blocked_by: []
created: 2026-02-27
---

# Create GPGparams.txt

Create `docker/GPGparams.txt` for unattended GPG key generation:

- No passphrase protection (`%no-protection`)
- RSA 2048 key + subkey
- Name: "Proton Mail Bridge Docker"
- Email: bridge@protonmail.docker
- No expiry
