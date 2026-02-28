---
type: task
title: "SEC-MEDIUM: GPG key has no passphrase"
status: open
priority: medium
labels: [security]
created: 2026-02-27
---

# GPG key has no passphrase

**Severity:** Medium

**Location:** `docker/GPGparams.txt:1` — `%no-protection`

**Description:** The GPG key used by `pass` (which stores bridge credentials) has no passphrase. Anyone with access to the volume mount can decrypt all stored passwords.

**Impact:** If the host's `./data` directory is readable by other users or processes, all bridge-stored credentials (Proton session tokens, bridge-generated passwords) are extractable.

**Recommendation:** This is an accepted trade-off for Docker (no interactive passphrase entry). Document it clearly in the README and ensure the volume mount has restrictive permissions. Add `chmod 700 /root` to the entrypoint.
