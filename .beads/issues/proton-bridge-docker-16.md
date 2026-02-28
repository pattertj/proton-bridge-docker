---
type: task
title: "Test Web UI end-to-end"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-13, proton-bridge-docker-14, proton-bridge-docker-15]
created: 2026-02-27
---

# Test Web UI end-to-end

Verify Phase 2 works:

1. Web UI loads at port 3000
2. Setup wizard appears for fresh container
3. Login flow works (including 2FA if applicable)
4. Sync progress displays correctly
5. Dashboard shows account credentials with working copy buttons
6. Container restart → dashboard loads with saved account (no re-login)
7. State transitions (RUNNING ↔ SETUP_MODE) work without deadlocks
