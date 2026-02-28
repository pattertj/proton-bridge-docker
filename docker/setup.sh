#!/bin/bash
echo "=== Proton Mail Bridge Setup ==="
echo "This will stop the running bridge and launch interactive CLI."
echo "After setup, the container will need to be restarted."
echo ""

# Kill running bridge to free the lock
pkill -f "bridge --noninteractive" || true
sleep 2

# Launch interactive CLI
/usr/bin/bridge --cli

echo ""
echo "Setup complete. Restart the container to apply changes."
echo "  docker restart <container_name>"
