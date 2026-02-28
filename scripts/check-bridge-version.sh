#!/bin/bash
# Compare local Dockerfile bridge version against latest upstream release
set -e

CURRENT=$(grep 'ARG BRIDGE_VERSION=' docker/Dockerfile | sed 's/.*=v//')
LATEST=$(curl -s https://api.github.com/repos/ProtonMail/proton-bridge/releases/latest | jq -r '.tag_name' | sed 's/^v//')

echo "Current: v${CURRENT}"
echo "Latest:  v${LATEST}"

if [ "$CURRENT" != "$LATEST" ]; then
    echo "Update available!"
    exit 1
else
    echo "Up to date."
    exit 0
fi
