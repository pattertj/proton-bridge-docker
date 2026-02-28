---
type: task
title: "SEC-HIGH: Container runs as root"
status: open
priority: high
labels: [security]
created: 2026-02-27
---

# Container runs as root

**Severity:** High

**Location:** `docker/Dockerfile` (no `USER` directive), `VOLUME /root`

**Description:** The container runs all processes (bridge, webui-server, socat) as root. The entire `/root` directory is the volume mount, meaning the host-mounted `./data` directory contains root-owned files.

**Impact:** If any process is compromised (e.g., via a bridge vulnerability), the attacker has root inside the container. Combined with the volume mount, this could allow writing to the host filesystem if Docker is misconfigured.

**Recommendation:** Create a dedicated non-root user. Set `HOME` to a custom path since bridge writes to `~/.config/protonmail` and `pass` expects `~/.gnupg`:

```dockerfile
RUN useradd -m -d /home/bridge bridge
USER bridge
WORKDIR /home/bridge
```
