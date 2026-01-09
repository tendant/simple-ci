#!/bin/bash
# Token Keeper - Maintains fresh Concourse token for gateway

CONCOURSE_URL="http://localhost:9001"
USERNAME="admin"
PASSWORD="admin"
TOKEN_FILE="/tmp/concourse-gateway-token"
FLY_TARGET="local"

echo "[$(date)] Starting Concourse Token Keeper"
echo "  Concourse URL: $CONCOURSE_URL"
echo "  Token file: $TOKEN_FILE"
echo "  Refresh interval: 12 hours"

# Function to refresh token
refresh_token() {
    echo "[$(date)] Refreshing Concourse token..."

    # Login via fly CLI
    fly -t "$FLY_TARGET" login -c "$CONCOURSE_URL" -u "$USERNAME" -p "$PASSWORD" -n 2>&1 | grep -E "target saved|logging in"

    if [ $? -eq 0 ]; then
        # Extract token from flyrc
        TOKEN=$(grep -A 2 "${FLY_TARGET}:" ~/.flyrc | grep "value:" | awk '{print $2}')

        if [ -n "$TOKEN" ]; then
            echo "$TOKEN" > "$TOKEN_FILE"
            chmod 600 "$TOKEN_FILE"
            echo "[$(date)] ✓ Token refreshed successfully (${#TOKEN} chars)"
            echo "[$(date)] Token file: $TOKEN_FILE"
            return 0
        else
            echo "[$(date)] ✗ ERROR: Failed to extract token from flyrc"
            return 1
        fi
    else
        echo "[$(date)] ✗ ERROR: Login failed"
        return 1
    fi
}

# Initial token fetch
refresh_token

# Keep refreshing every 12 hours
while true; do
    echo "[$(date)] Sleeping for 12 hours until next refresh..."
    sleep 43200  # 12 hours

    refresh_token
done
