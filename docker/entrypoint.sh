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

# --- 2. D-Bus session bus (required by bridge keychain) ---
if [ -z "$DBUS_SESSION_BUS_ADDRESS" ]; then
    eval "$(dbus-launch --sh-syntax)"
    export DBUS_SESSION_BUS_ADDRESS
fi

# --- 3. Environment Variables ---
SMTP_PORT="${CONTAINER_SMTP_PORT:-25}"
IMAP_PORT="${CONTAINER_IMAP_PORT:-143}"
WEBUI_PORT="${WEBUI_PORT:-3000}"

# --- 4. Start socat proxies ---
# Bridge binds to 127.0.0.1 only. socat forwards from 0.0.0.0 (container interface)
# so other containers/hosts on the network can reach the bridge.
echo "[entrypoint] Starting socat proxies..."
socat TCP-LISTEN:${SMTP_PORT},fork,reuseaddr TCP:127.0.0.1:1025 &
socat TCP-LISTEN:${IMAP_PORT},fork,reuseaddr TCP:127.0.0.1:1143 &

# --- 5. Start Web UI server ---
echo "[entrypoint] Starting Web UI on port ${WEBUI_PORT}..."
/usr/bin/webui-server -port="${WEBUI_PORT}" &

# --- 6. Start Bridge ---
echo "[entrypoint] Starting Proton Mail Bridge..."
/usr/bin/bridge --noninteractive &

# Forward signals to children so `docker stop` works cleanly
trap 'kill $(jobs -p) 2>/dev/null; wait' SIGTERM SIGINT

# Keep container alive — PID 1 is bash, not bridge,
# so the webui can kill/restart bridge without crashing the container.
wait
