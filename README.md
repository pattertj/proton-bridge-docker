# Proton Mail Bridge Docker

[![Build](https://img.shields.io/github/actions/workflow/status/pattertj/proton-bridge-docker/build-and-push.yml?label=Build)](https://github.com/pattertj/proton-bridge-docker/actions/workflows/build-and-push.yml)
[![Release](https://img.shields.io/github/actions/workflow/status/pattertj/proton-bridge-docker/release.yml?label=Release)](https://github.com/pattertj/proton-bridge-docker/actions/workflows/release.yml)
[![Upstream Check](https://img.shields.io/github/actions/workflow/status/pattertj/proton-bridge-docker/check-upstream.yml?label=Upstream%20Check)](https://github.com/pattertj/proton-bridge-docker/actions/workflows/check-upstream.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/pattertj/protonmail-bridge)](https://hub.docker.com/r/pattertj/protonmail-bridge)
[![Docker Image Size](https://img.shields.io/docker/image-size/pattertj/protonmail-bridge/latest)](https://hub.docker.com/r/pattertj/protonmail-bridge)
[![GHCR](https://img.shields.io/badge/ghcr.io-pattertj%2Fprotonmail--bridge-blue?logo=github)](https://ghcr.io/pattertj/protonmail-bridge)
[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

Docker container for [Proton Mail Bridge](https://proton.me/mail/bridge) with a web-based setup UI, optimized for Unraid.

Creates a local IMAP/SMTP server that decrypts your Proton Mail, allowing other containers and email clients on your network to send and receive email through your Proton account.

## Features

- **Web UI** for account setup and credential display -- no `docker exec` required
- **Multi-arch** images (amd64, arm64)
- **Automated updates** via CI that tracks upstream Proton Bridge releases
- **Unraid Community Applications** template for one-click install
- **Security hardened**: auth tokens, rate limiting, input sanitization, security headers

## Requirements

- A **paid Proton plan** (Mail Plus, Proton Unlimited, or Proton for Business). Free accounts cannot use Bridge.

## Quick Start

```yaml
services:
  protonmail-bridge:
    image: pattertj/protonmail-bridge:latest
    container_name: protonmail-bridge
    restart: unless-stopped
    volumes:
      - ./data:/root
    ports:
      - "1025:25"     # SMTP
      - "1143:143"    # IMAP
      - "3080:3000"   # Web UI
    environment:
      - WEBUI_AUTH_TOKEN=your-secret-token
```

1. Start the container: `docker compose up -d`
2. Open the Web UI: `http://localhost:3080`
3. Enter your `WEBUI_AUTH_TOKEN` to authenticate
4. Sign in with your Proton credentials
5. Use the displayed IMAP/SMTP settings in your apps

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBUI_AUTH_TOKEN` | *(empty)* | Access token for the Web UI. **Set this.** |
| `WEBUI_PORT` | `3000` | Internal web UI port |
| `WEBUI_TLS_CERT` | *(empty)* | Path to TLS certificate for HTTPS |
| `WEBUI_TLS_KEY` | *(empty)* | Path to TLS private key for HTTPS |
| `CONTAINER_SMTP_PORT` | `25` | Internal SMTP listen port |
| `CONTAINER_IMAP_PORT` | `143` | Internal IMAP listen port |

## Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 25 | SMTP | Email sending |
| 143 | IMAP | Email receiving |
| 3000 | HTTP(S) | Web UI |

## Volumes

| Path | Description |
|------|-------------|
| `/root` | Bridge config, GPG keyring, password store, mail cache. **Must persist across recreates.** |

## Using with Other Containers

Point your apps at the bridge container's hostname/IP with the mapped ports:

| Setting | Value |
|---------|-------|
| SMTP Host | `protonmail-bridge` (or container IP) |
| SMTP Port | `1025` (your mapped port) |
| IMAP Host | `protonmail-bridge` (or container IP) |
| IMAP Port | `1143` (your mapped port) |
| Username | Shown in Web UI after login |
| Password | Bridge-generated password shown in Web UI |
| Security | STARTTLS |

Works with Nextcloud, Gitea, Home Assistant, Uptime Kuma, Paperless-ngx, and any service that supports SMTP/IMAP.

## Building from Source

```bash
docker compose up -d --build
```

### Running Tests

```bash
cd webui && go test ./handlers/ -v
bash tests/test_entrypoint.sh
bash tests/test_no_cdn.sh
```

## Security

- Set `WEBUI_AUTH_TOKEN` to protect the Web UI
- Use `WEBUI_TLS_CERT`/`WEBUI_TLS_KEY` for HTTPS on untrusted networks
- The bridge-generated password (not your Proton password) is used by email clients
- Credentials are stored in an encrypted GPG/pass keychain inside the container volume

## License

GPL-3.0 -- see [LICENSE](LICENSE)
