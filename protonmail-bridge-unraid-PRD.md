# PRD: Proton Mail Bridge Docker for Unraid

**Project:** `protonmail-bridge-unraid`
**Author:** Tyler Patterson
**Date:** February 2026
**Status:** Implementation Ready

---

## 1. Overview

Build a Docker container for Proton Mail Bridge optimized for Unraid, with a lightweight web UI for setup and status monitoring, published to the Unraid Community Applications store. This replaces the abandoned `shenxn/protonmail-bridge-docker` (573 stars, unmaintained) and improves upon `VideoCurio/ProtonMailBridgeDocker` by adding a web-based setup experience instead of requiring CLI `docker exec` interactions.

### 1.1 Goals

- Fully functional Proton Mail Bridge container exposing IMAP (143) and SMTP (25) to other containers and local network clients
- Web UI for account setup, status monitoring, and credential display — no `docker exec` required for normal usage
- Automated CI/CD that tracks Proton Bridge releases and publishes updated images
- Unraid Community Applications template for one-click install
- Multi-architecture support (amd64 primary, arm64 secondary)

### 1.2 Non-Goals

- Full email client functionality
- Support for free-tier Proton accounts (Bridge requires a paid plan)
- Custom TLS certificate management in v1 (may add later)
- Calendar or contacts sync (Bridge only handles IMAP/SMTP)

---

## 2. Repository Structure

```
protonmail-bridge-unraid/
├── docker/
│   ├── Dockerfile                  # Multi-stage build
│   ├── entrypoint.sh               # Container entrypoint
│   ├── setup.sh                    # Fallback CLI setup script
│   └── GPGparams.txt               # GPG key generation template
├── webui/
│   ├── main.go                     # Go HTTP server + bridge CLI automation
│   ├── go.mod
│   ├── go.sum
│   ├── handlers/
│   │   ├── api.go                  # REST API handlers
│   │   ├── bridge.go               # Bridge CLI interaction layer
│   │   └── status.go               # Health/status endpoints
│   └── frontend/
│       ├── index.html              # Single-page app (embedded in Go binary)
│       ├── app.js                  # React app (bundled, no build step needed)
│       └── styles.css
├── unraid/
│   ├── protonmail-bridge.xml       # Unraid CA template
│   └── ca_profile.xml             # Maintainer profile
├── .github/
│   └── workflows/
│       ├── build-and-push.yml      # Build on push/tag
│       ├── check-upstream.yml      # Weekly upstream version check
│       └── release.yml             # Create release on new bridge version
├── scripts/
│   └── check-bridge-version.sh     # Compare upstream version
├── README.md
├── LICENSE                         # GPL-3.0 (matches Bridge license)
└── docker-compose.yml              # Example compose file
```

---

## 3. Docker Image

### 3.1 Dockerfile (Multi-stage Build)

**Stage 1: Build Proton Bridge from source**

```dockerfile
FROM golang:1.24-bookworm AS bridge-builder

ARG BRIDGE_VERSION=v3.22.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    git gcc libc6-dev libsecret-1-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build
RUN git clone --depth 1 --branch ${BRIDGE_VERSION} \
    https://github.com/ProtonMail/proton-bridge.git .

# Build CLI-only (no Qt/GUI dependencies)
RUN make build-nogui
```

**Stage 2: Build Web UI**

```dockerfile
FROM golang:1.24-bookworm AS webui-builder

WORKDIR /webui
COPY webui/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /webui-server .
```

**Stage 3: Runtime**

```dockerfile
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    pass \
    gnupg \
    socat \
    ca-certificates \
    procps \
    && rm -rf /var/lib/apt/lists/*

COPY --from=bridge-builder /build/bridge /usr/bin/bridge
COPY --from=webui-builder /webui-server /usr/bin/webui-server
COPY docker/entrypoint.sh /app/entrypoint.sh
COPY docker/setup.sh /app/setup.sh
COPY docker/GPGparams.txt /app/GPGparams.txt

RUN chmod +x /app/entrypoint.sh /app/setup.sh

EXPOSE 25 143 3000

VOLUME /root

LABEL org.opencontainers.image.title="Proton Mail Bridge" \
      org.opencontainers.image.description="Proton Mail IMAP/SMTP Bridge with Web UI for Unraid" \
      org.opencontainers.image.source="https://github.com/YOURUSER/protonmail-bridge-unraid" \
      org.opencontainers.image.licenses="GPL-3.0"

ENTRYPOINT ["/app/entrypoint.sh"]
```

### 3.2 Entrypoint Script (`docker/entrypoint.sh`)

The entrypoint must handle the following concerns in order:

```bash
#!/bin/bash
set -e

# --- 1. GPG + Pass Initialization ---
# Bridge requires a keychain. In Docker, we use `pass` backed by GPG.
# Only initialize if not already set up (persisted in /root volume).

if [ ! -d /root/.gnupg ] || [ -z "$(gpg --list-keys 2>/dev/null)" ]; then
    echo "[entrypoint] Initializing GPG keyring..."
    gpg --batch --gen-key /app/GPGparams.txt
fi

GPG_KEY_ID=$(gpg --list-keys --keyid-format long 2>/dev/null | grep -A1 "^pub" | tail -1 | xargs)

if [ ! -d /root/.password-store ]; then
    echo "[entrypoint] Initializing pass store..."
    pass init "$GPG_KEY_ID"
fi

# --- 2. Environment Variables ---
SMTP_PORT="${CONTAINER_SMTP_PORT:-25}"
IMAP_PORT="${CONTAINER_IMAP_PORT:-143}"
WEBUI_PORT="${WEBUI_PORT:-3000}"

# --- 3. Start socat proxies ---
# Bridge binds to 127.0.0.1 only. socat forwards from 0.0.0.0 (container interface)
# so other containers/hosts on the network can reach the bridge.
echo "[entrypoint] Starting socat proxies..."
socat TCP-LISTEN:${SMTP_PORT},fork,reuseaddr TCP:127.0.0.1:1025 &
socat TCP-LISTEN:${IMAP_PORT},fork,reuseaddr TCP:127.0.0.1:1143 &

# --- 4. Start Web UI server ---
echo "[entrypoint] Starting Web UI on port ${WEBUI_PORT}..."
/usr/bin/webui-server -port="${WEBUI_PORT}" &

# --- 5. Start Bridge ---
# Run bridge in non-interactive mode. The --cli flag is for interactive use.
# For daemon mode, bridge runs without --cli and stays in foreground.
echo "[entrypoint] Starting Proton Mail Bridge..."
exec /usr/bin/bridge --noninteractive
```

### 3.3 GPG Parameters (`docker/GPGparams.txt`)

```
%no-protection
Key-Type: RSA
Key-Length: 2048
Subkey-Type: RSA
Subkey-Length: 2048
Name-Real: Proton Mail Bridge Docker
Name-Email: bridge@protonmail.docker
Expire-Date: 0
%commit
```

### 3.4 Fallback Setup Script (`docker/setup.sh`)

For advanced users or debugging, provide a CLI-based setup path:

```bash
#!/bin/bash
echo "=== Proton Mail Bridge Setup ==="
echo "This will stop the running bridge and launch interactive CLI."
echo "After setup, the container will need to be restarted."
echo ""

# Kill running bridge to free the lock
pkill -f "bridge" || true
sleep 2

# Launch interactive CLI
/usr/bin/bridge --cli

echo ""
echo "Setup complete. Restart the container to apply changes."
echo "  docker restart <container_name>"
```

Usage: `docker exec -it protonmail-bridge /app/setup.sh`

---

## 4. Web UI

### 4.1 Architecture

A single Go binary serves both the REST API and the embedded static frontend. The Go server interacts with the Bridge CLI by spawning `bridge --cli` as a child process and automating input/output via stdin/stdout pipes.

### 4.2 Go Server (`webui/main.go`)

```
webui/
├── main.go              # HTTP server, router, embedded static files
├── go.mod
├── handlers/
│   ├── api.go           # REST endpoint handlers
│   ├── bridge.go        # Bridge CLI process management
│   └── status.go        # Status and health checks
└── frontend/
    ├── index.html        # SPA shell
    ├── app.js            # React app (CDN-loaded React, no build tooling)
    └── styles.css        # Minimal styling
```

### 4.3 API Endpoints

```
GET  /api/status          → Bridge status (running, syncing, accounts, etc.)
GET  /api/accounts        → List configured accounts with IMAP/SMTP credentials
POST /api/login           → Initiate login flow
POST /api/login/2fa       → Submit 2FA code (second step of login)
POST /api/accounts/:id/mode → Toggle combined/split address mode
POST /api/logout/:id      → Remove account
GET  /api/health          → Simple healthcheck for monitoring
GET  /                    → Serves embedded frontend SPA
```

### 4.4 Bridge CLI Interaction Layer (`handlers/bridge.go`)

This is the core technical challenge. The bridge CLI is an interactive REPL. The Go server manages it as follows:

```go
// BridgeCLI manages a bridge --cli subprocess
type BridgeCLI struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    mu     sync.Mutex  // serialize CLI commands
}

// NewBridgeCLI spawns bridge --cli and manages I/O
func NewBridgeCLI() (*BridgeCLI, error) {
    // 1. Kill the running non-interactive bridge (only one instance allowed)
    // 2. Spawn: bridge --cli
    // 3. Wait for ">>>" prompt
    // 4. Return ready BridgeCLI
}

// Execute sends a command and reads until next ">>>" prompt
func (b *BridgeCLI) Execute(command string) (string, error) {
    b.mu.Lock()
    defer b.mu.Unlock()
    // Write command + \n to stdin
    // Read stdout until ">>>" appears
    // Return captured output
}

// Login handles the multi-step login flow
func (b *BridgeCLI) Login(username, password string) (LoginResult, error) {
    // 1. Send "login"
    // 2. Wait for "Username:" prompt, send username
    // 3. Wait for "Password:" prompt, send password
    // 4. Check for "Two factor code:" prompt
    //    - If present, return NeedsTwoFactor status
    //    - If not, return success/failure
}

// SubmitTwoFactor completes the 2FA step
func (b *BridgeCLI) SubmitTwoFactor(code string) (LoginResult, error) {
    // Send the 2FA code
    // Parse result for success/failure
}

// GetAccountInfo returns parsed output of "info" command
func (b *BridgeCLI) GetAccountInfo() ([]Account, error) {
    // Send "info"
    // Parse structured output into Account structs:
    //   - Email, IMAP port, SMTP port, Username, Password, Security
}

// Close cleanly shuts down the CLI subprocess
func (b *BridgeCLI) Close() error {
    // Send "exit"
    // Wait for process to terminate
}
```

**Critical implementation notes for the CLI interaction layer:**

1. **Only one bridge instance can run at a time.** The web UI server must kill the non-interactive bridge, start CLI mode for the operation, then after the operation completes, kill CLI mode and restart non-interactive mode. Use a state machine: `RUNNING` → `SETUP_MODE` → `RUNNING`.

2. **Prompt detection:** The bridge CLI uses `>>> ` as its prompt. Read stdout in a goroutine, buffer output, and signal when the prompt appears. Use a timeout (30s) to prevent hanging.

3. **Login is multi-step and may fail.** Parse output for known patterns:
   - `"Username:"` → send username
   - `"Password:"` → send password
   - `"Two factor code:"` → return to frontend, request 2FA
   - `"Authenticating"` → wait for result
   - `"was added successfully"` → success
   - `"Incorrect login credentials"` → auth failure

4. **Sync progress:** After login, the bridge syncs the mailbox. Parse lines matching `Sync (account): XX.X%` and expose via the status endpoint for the frontend progress bar.

5. **Thread safety:** All CLI interactions must be serialized. The `sync.Mutex` in `BridgeCLI` ensures this.

### 4.5 Frontend Design

The frontend is a single-page React app loaded via CDN (React, ReactDOM from cdnjs). No build tooling. The Go server embeds and serves the HTML/JS/CSS files using `embed.FS`.

**States/Views:**

1. **No Account Configured (Setup Wizard)**
   - Clean form: Email input, Password input (masked), Submit button
   - On submit → `POST /api/login`
   - If 2FA required → show 2FA code input
   - On success → transition to Account View with sync progress

2. **Syncing**
   - Progress bar with percentage
   - Estimated time remaining
   - Account name

3. **Running (Main Dashboard)**
   - Green status indicator: "Bridge is running"
   - For each account:
     - Account email
     - IMAP settings: host, port, username, generated password (with copy button)
     - SMTP settings: host, port, username, generated password (with copy button)
     - Security: STARTTLS
   - Note about ports: "Use the host ports configured in your Docker/Unraid settings, not the internal ports shown by Bridge."
   - Button: "Add Another Account"
   - Button: "Toggle Split/Combined Mode"
   - Button: "Remove Account"

4. **Error State**
   - Bridge not running / crashed
   - Login failed (bad credentials, account locked, etc.)
   - Sync failed

**Styling:** Keep it minimal and clean. Use Proton's purple (#6D4AFF) as the accent color. No heavy frameworks — plain CSS or Tailwind via CDN. The Unraid WebUI link opens in an iframe or new tab, so the UI should work well standalone at any size.

**Key UI behaviors:**

- The IMAP/SMTP password is the bridge-generated password, NOT the user's Proton password. Make this very clear in the UI with a callout/note.
- Include a "Copy All Settings" button that copies a formatted block of connection settings to clipboard.
- The host shown should be a helpful note: "Use this container's IP or hostname on your Docker network."
- Poll `/api/status` every 5 seconds when on the dashboard to update sync status and connection state.

### 4.6 State Persistence

The web UI server must handle container restarts gracefully. On startup:

1. Check if `/root/.config/protonmail/bridge/` exists (bridge config directory)
2. If accounts are configured, start bridge in non-interactive mode, serve dashboard
3. If no accounts configured, serve the setup wizard

Store a minimal state file at `/root/.webui-state.json`:

```json
{
  "setup_complete": true,
  "last_bridge_version": "3.22.0",
  "accounts": ["user@proton.me"]
}
```

---

## 5. Unraid Template

### 5.1 CA Template (`unraid/protonmail-bridge.xml`)

```xml
<?xml version="1.0"?>
<Container version="2">
  <Name>ProtonMailBridge</Name>
  <Repository>YOURUSER/protonmail-bridge:latest</Repository>
  <Registry>https://hub.docker.com/r/YOURUSER/protonmail-bridge</Registry>
  <Branch>
    <Tag>latest</Tag>
    <TagDescription>Latest stable release (Debian-based)</TagDescription>
  </Branch>
  <Branch>
    <Tag>alpine</Tag>
    <TagDescription>Lightweight Alpine-based image</TagDescription>
  </Branch>
  <Network>bridge</Network>
  <Shell>bash</Shell>
  <Privileged>false</Privileged>
  <Support>https://forums.unraid.net/topic/XXXXX-protonmail-bridge-docker/</Support>
  <Project>https://github.com/YOURUSER/protonmail-bridge-unraid</Project>
  <Overview>
Proton Mail Bridge creates a local IMAP/SMTP server that decrypts your Proton Mail, allowing other containers and email clients on your network to send and receive email through your Proton account.

Includes a web-based setup UI — no command line required. After installing, open the WebUI to log in with your Proton credentials and view your IMAP/SMTP connection details.

IMPORTANT: Requires a paid Proton plan (Mail Plus, Proton Unlimited, or Proton for Business). Free accounts cannot use Bridge.

Use cases: Send notifications from apps like Nextcloud, Gitea, Home Assistant, Uptime Kuma, Paperless-ngx, or any service that supports SMTP. Receive mail via IMAP in self-hosted mail clients.
  </Overview>
  <Category>Network:Other Productivity:</Category>
  <WebUI>http://[IP]:[PORT:3000]</WebUI>
  <Icon>https://raw.githubusercontent.com/YOURUSER/protonmail-bridge-unraid/main/unraid/icon.png</Icon>
  <ExtraParams/>
  <PostArgs/>
  <DonateText>Buy me a coffee</DonateText>
  <DonateLink>https://buymeacoffee.com/YOURUSER</DonateLink>
  <DonateImg/>
  <Requires/>
  <Config
    Name="Config Storage"
    Target="/root"
    Default="/mnt/user/appdata/protonmail-bridge"
    Mode="rw"
    Description="Persistent storage for Bridge configuration, credentials, and mail cache. Must survive container recreates."
    Type="Path"
    Display="always"
    Required="true"
    Mask="false">/mnt/user/appdata/protonmail-bridge</Config>
  <Config
    Name="SMTP Port"
    Target="25"
    Default="1025"
    Mode="tcp"
    Description="SMTP port for sending email. Point your apps here. Default: 1025"
    Type="Port"
    Display="always"
    Required="true"
    Mask="false">1025</Config>
  <Config
    Name="IMAP Port"
    Target="143"
    Default="1143"
    Mode="tcp"
    Description="IMAP port for receiving email. Default: 1143"
    Type="Port"
    Display="always"
    Required="true"
    Mask="false">1143</Config>
  <Config
    Name="WebUI Port"
    Target="3000"
    Default="3080"
    Mode="tcp"
    Description="Web interface for setup and monitoring. Default: 3080"
    Type="Port"
    Display="always"
    Required="true"
    Mask="false">3080</Config>
  <Config
    Name="WEBUI_PORT"
    Target="WEBUI_PORT"
    Default="3000"
    Description="Internal web UI port (do not change unless you have a conflict)"
    Type="Variable"
    Display="advanced"
    Required="false"
    Mask="false">3000</Config>
</Container>
```

### 5.2 Maintainer Profile (`unraid/ca_profile.xml`)

```xml
<?xml version="1.0" encoding="utf-8"?>
<Maintainer>
  <Icon>https://raw.githubusercontent.com/YOURUSER/protonmail-bridge-unraid/main/unraid/profile.png</Icon>
  <Name>Tyler Patterson</Name>
  <Overview>Self-hosted privacy tools for Unraid.</Overview>
  <GitHub>https://github.com/YOURUSER</GitHub>
</Maintainer>
```

### 5.3 Icon Requirements

Create or source a clean PNG icon for the app. Recommended: 512x512 PNG with transparent background. Use the Proton Mail Bridge logo concept (a bridge icon with the Proton purple). Host in the repo under `unraid/icon.png` and reference via raw GitHub URL.

---

## 6. GitHub Actions

### 6.1 Build and Push (`build-and-push.yml`)

Triggered on pushes to `main` and version tags. Builds multi-arch images and pushes to Docker Hub and GHCR.

```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

env:
  DOCKER_HUB_REPO: YOURUSER/protonmail-bridge
  GHCR_REPO: ghcr.io/YOURUSER/protonmail-bridge

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Docker Hub
        if: github.event_name != 'pull_request'
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Login to GHCR
        if: github.event_name != 'pull_request'
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: |
            ${{ env.DOCKER_HUB_REPO }}
            ${{ env.GHCR_REPO }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,prefix=
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          file: docker/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Update Docker Hub description
        if: github.ref == 'refs/heads/main'
        uses: peter-evans/dockerhub-description@v4
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
          repository: ${{ env.DOCKER_HUB_REPO }}
          readme-filepath: ./README.md
```

### 6.2 Upstream Version Check (`check-upstream.yml`)

Runs weekly (and on manual trigger). Checks for new Proton Bridge releases. If found, opens a PR to bump the version.

```yaml
name: Check Upstream Bridge Version

on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9am UTC
  workflow_dispatch:       # Manual trigger

jobs:
  check-version:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Get current version
        id: current
        run: |
          CURRENT=$(grep 'ARG BRIDGE_VERSION=' docker/Dockerfile | sed 's/.*=v//')
          echo "version=${CURRENT}" >> $GITHUB_OUTPUT

      - name: Get latest upstream version
        id: upstream
        run: |
          LATEST=$(curl -s https://api.github.com/repos/ProtonMail/proton-bridge/releases/latest | jq -r '.tag_name' | sed 's/^v//')
          echo "version=${LATEST}" >> $GITHUB_OUTPUT

      - name: Compare versions
        id: compare
        run: |
          if [ "${{ steps.current.outputs.version }}" != "${{ steps.upstream.outputs.version }}" ]; then
            echo "update_available=true" >> $GITHUB_OUTPUT
            echo "New version available: ${{ steps.upstream.outputs.version }} (current: ${{ steps.current.outputs.version }})"
          else
            echo "update_available=false" >> $GITHUB_OUTPUT
            echo "Already on latest: ${{ steps.current.outputs.version }}"
          fi

      - name: Update Dockerfile
        if: steps.compare.outputs.update_available == 'true'
        run: |
          NEW_VERSION="${{ steps.upstream.outputs.version }}"
          sed -i "s/ARG BRIDGE_VERSION=v.*/ARG BRIDGE_VERSION=v${NEW_VERSION}/" docker/Dockerfile
          # Also update VERSION file if it exists
          echo "${NEW_VERSION}" > VERSION

      - name: Create Pull Request
        if: steps.compare.outputs.update_available == 'true'
        uses: peter-evans/create-pull-request@v6
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore: bump Proton Bridge to v${{ steps.upstream.outputs.version }}"
          title: "Bump Proton Bridge to v${{ steps.upstream.outputs.version }}"
          body: |
            Automated version bump detected a new Proton Mail Bridge release.

            **Current:** v${{ steps.current.outputs.version }}
            **New:** v${{ steps.upstream.outputs.version }}

            [Release notes](https://github.com/ProtonMail/proton-bridge/releases/tag/v${{ steps.upstream.outputs.version }})

            After merging, tag the commit to trigger a release build:
            ```
            git tag v${{ steps.upstream.outputs.version }}
            git push origin v${{ steps.upstream.outputs.version }}
            ```
          branch: "auto/bridge-v${{ steps.upstream.outputs.version }}"
          labels: |
            automated
            dependency-update
```

### 6.3 Release Workflow (`release.yml`)

Creates a GitHub Release when a version tag is pushed. The `build-and-push.yml` handles the Docker image building on tags automatically.

```yaml
name: Create Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Get Bridge version from Dockerfile
        id: bridge_version
        run: |
          VERSION=$(grep 'ARG BRIDGE_VERSION=' docker/Dockerfile | sed 's/.*=v//')
          echo "version=${VERSION}" >> $GITHUB_OUTPUT

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          name: "v${{ steps.bridge_version.outputs.version }}"
          body: |
            ## Proton Mail Bridge v${{ steps.bridge_version.outputs.version }}

            Docker images:
            - `docker pull YOURUSER/protonmail-bridge:${{ github.ref_name }}`
            - `docker pull YOURUSER/protonmail-bridge:latest`
            - `docker pull ghcr.io/YOURUSER/protonmail-bridge:${{ github.ref_name }}`

            [Upstream release notes](https://github.com/ProtonMail/proton-bridge/releases/tag/v${{ steps.bridge_version.outputs.version }})
          generate_release_notes: true
```

---

## 7. Docker Compose Example

```yaml
services:
  protonmail-bridge:
    image: YOURUSER/protonmail-bridge:latest
    container_name: protonmail-bridge
    restart: unless-stopped
    volumes:
      - ./data:/root
    ports:
      - "1025:25"     # SMTP
      - "1143:143"    # IMAP
      - "3080:3000"   # Web UI
    environment:
      - WEBUI_PORT=3000
```

---

## 8. Implementation Order

Execute in this sequence. Each phase should be working and testable before proceeding to the next.

### Phase 1: Docker Foundation (Get bridge running in a container)

1. Create `docker/Dockerfile` — multi-stage build, bridge from source, runtime with socat/pass/gnupg
2. Create `docker/entrypoint.sh` — GPG init, pass init, socat proxies, bridge launch
3. Create `docker/GPGparams.txt`
4. Create `docker-compose.yml`
5. Create `docker/setup.sh` — fallback CLI setup
6. **Test:** `docker compose up`, then `docker exec -it protonmail-bridge /app/setup.sh`, login, verify IMAP/SMTP work from another container or `telnet`

### Phase 2: Web UI (Eliminate the docker exec requirement)

1. Create `webui/go.mod`, `webui/main.go` — HTTP server skeleton with embedded static files
2. Create `webui/handlers/bridge.go` — CLI interaction layer (the hard part)
   - Start with just `Execute()`, `GetAccountInfo()`
   - Then add `Login()`, `SubmitTwoFactor()`
   - Test each operation manually before wiring to API
3. Create `webui/handlers/api.go` — REST endpoints
4. Create `webui/handlers/status.go` — health and status
5. Create `webui/frontend/` — React SPA
   - Setup wizard (login form → 2FA form → sync progress)
   - Dashboard (account info, credentials, status)
6. Update `docker/Dockerfile` to include webui build stage
7. Update `docker/entrypoint.sh` to launch webui-server
8. **Test:** Full flow through web UI — login, view credentials, restart container, verify dashboard loads with saved account

### Phase 3: CI/CD

1. Create `.github/workflows/build-and-push.yml`
2. Create `.github/workflows/check-upstream.yml`
3. Create `.github/workflows/release.yml`
4. Set up Docker Hub repo and GHCR
5. Configure GitHub secrets (DOCKERHUB_USERNAME, DOCKERHUB_TOKEN)
6. **Test:** Push to main, verify image builds. Manually trigger version check.

### Phase 4: Unraid Publishing

1. Create `unraid/protonmail-bridge.xml`
2. Create `unraid/ca_profile.xml`
3. Create/source icon PNG
4. Create Unraid forum support thread (Docker Containers subforum)
5. Submit to Community Applications via their submission form
6. **Test:** Install from CA on actual Unraid hardware, verify full WebUI flow works, verify SMTP/IMAP accessible from other Unraid containers

---

## 9. Technical Risks and Mitigations

### 9.1 Bridge CLI Output Parsing

**Risk:** Proton can change CLI output format between versions, breaking the parser.

**Mitigation:** Build the parser with flexible pattern matching, not strict position-based parsing. Log all CLI I/O at debug level. Include integration tests that run against the actual bridge CLI. The version check workflow gives us a heads-up when new versions land — always test against the new version before merging the bump PR.

### 9.2 Bridge Only Allows One Instance

**Risk:** The web UI needs to spawn CLI mode for login, but the non-interactive bridge is already running.

**Mitigation:** Implement a state machine in the web UI server:
- `RUNNING` → bridge is running non-interactively, web UI shows dashboard
- `SETUP_MODE` → bridge is stopped, CLI is active, web UI shows setup wizard
- `TRANSITIONING` → brief period while switching modes

The transition takes ~2-5 seconds. Show a "Entering setup mode..." spinner in the UI. IMAP/SMTP will be briefly unavailable during this window — document this clearly.

### 9.3 socat Reliability

**Risk:** socat proxy processes can die, silently breaking SMTP/IMAP connectivity.

**Mitigation:** The entrypoint should monitor socat processes. A simple approach: run socat with the `fork` option (it spawns a new process per connection, main socat stays alive). Add a healthcheck in the webui-server that periodically verifies socat PIDs are running and restarts them if not.

### 9.4 Credentials Security

**Risk:** Proton credentials are entered via the web UI, transmitted over HTTP within the Docker network.

**Mitigation:** Document that the web UI should only be accessed on trusted networks (same as any Unraid management interface). The credentials are only used transiently during the login flow — they are not stored by the web UI. After login, only the bridge-generated IMAP/SMTP passwords are displayed, which are separate from the Proton account password. For v2, consider adding optional HTTPS support with self-signed certs.

### 9.5 Unraid CA Rejection

**Risk:** The CA moderation team may reject the submission if the template doesn't meet their standards, or if they consider it a duplicate of the existing broken entry.

**Mitigation:** The existing shenxn entry is unmaintained and broken — this is a replacement, not a duplicate. Emphasize the web UI differentiation and active maintenance commitment. Create a thorough support thread with setup documentation before submitting. Follow the template XML schema exactly as documented (generate with dockerMan if possible, don't hand-edit).

---

## 10. Future Enhancements (Post-v1)

These are explicitly out of scope for v1 but worth tracking:

- **Optional HTTPS for web UI** with self-signed or user-provided certs
- **Alpine-based image variant** for smaller footprint
- **Webhook notifications** when bridge disconnects or auth expires
- **SMTP relay testing** button in the web UI (send a test email)
- **Multiple account management** improvements in the UI
- **gRPC integration** instead of CLI automation for bridge control (more robust, but coupled to bridge internals)
- **Auto-login on container restart** if stored credentials are valid
- **Prometheus metrics endpoint** for monitoring integration
