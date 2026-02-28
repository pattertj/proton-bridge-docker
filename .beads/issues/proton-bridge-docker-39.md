---
type: task
title: "SEC-LOW: GET /api/accounts triggers bridge restart (DoS)"
status: done
priority: medium
labels: [security]
created: 2026-02-27
---

# GET /api/accounts triggers bridge restart (DoS)

**Severity:** Low (escalates to Medium combined with no auth)

**Location:** `webui/handlers/api.go:34-71`

**Description:** `GET /api/accounts` calls `EnterSetupMode()` which kills the running bridge process, then `ExitSetupMode()` restarts it. A simple GET request causes a service disruption (IMAP/SMTP go down for ~4-6 seconds).

**Impact:** Denial of service — any unauthenticated request to `/api/accounts` interrupts all email connectivity. Combined with no auth (issue #26), this is remotely exploitable by anyone on the network.

**Recommendation:** Cache account info from the last setup session. Only enter setup mode when explicitly requested by the user via a dedicated `POST /api/accounts/refresh` endpoint. The `GET /api/accounts` should return cached data only.
