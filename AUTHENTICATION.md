# Concourse Authentication Guide

## The Challenge

Concourse CI uses OAuth2 for authentication, which presents challenges for long-running services like our gateway:
- Tokens expire (typically 24 hours)
- Manual token extraction from fly CLI isn't sustainable
- Need automatic token refresh

## Current Situation

Your Concourse is started with local authentication:
```bash
--add-local-user admin:admin
--main-team-local-user admin
```

This means `admin` user has full permissions on the `main` team.

## Token Expiration

**Fly CLI tokens**: Expire after ~24 hours
**OAuth tokens**: Configurable, but typically short-lived (hours to days)

## Reliable Solutions

### Option 1: Token Refresh Flow (Recommended for Production)

The gateway should automatically refresh tokens before expiry.

**Implementation Steps:**

1. **Update Token Manager** to properly handle OAuth token refresh:

```go
// In internal/provider/concourse/auth.go

// GetToken with automatic refresh
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
    tm.mu.RLock()
    // Check if token is still valid
    if tm.token != "" && time.Now().Before(tm.tokenExpiry.Add(-tm.refreshMargin)) {
        token := tm.token
        tm.mu.RUnlock()
        return token, nil
    }
    tm.mu.RUnlock()

    // Token expired or doesn't exist - fetch new one
    return tm.refreshToken(ctx)
}
```

2. **Fix the OAuth Scope Issue**:

The issue is that basic `openid` scope doesn't grant API permissions. We need to investigate Concourse's OAuth configuration.

**Check your Concourse OAuth settings:**
```bash
# Check if Concourse has OAuth providers configured
curl http://localhost:9001/sky/issuer/.well-known/openid-configuration
```

### Option 2: Extract and Use Fly Token (Quick Fix)

For development/testing, extract fly's token:

**Get Token:**
```bash
# Method 1: From flyrc
grep -A 2 "local:" ~/.flyrc | grep "value:" | awk '{print $2}'

# Method 2: Via fly command
fly -t local curl /api/v1/teams/main --print-and-exit | grep "Authorization:" | awk '{print $3}'
```

**Token Lifespan**: ~24 hours (will need manual refresh)

**Update config:**
```yaml
concourse:
  bearer_token: "YOUR_TOKEN_HERE"
```

**Token Refresh Script:**
```bash
#!/bin/bash
# scripts/refresh-concourse-token.sh

# Login to get fresh token
fly -t local login -c http://localhost:9001 -u admin -p admin

# Extract token
TOKEN=$(grep -A 2 "local:" ~/.flyrc | grep "value:" | awk '{print $2}')

# Update config
sed -i.bak "s/bearer_token: .*/bearer_token: \"$TOKEN\"/" configs/gateway.yaml

# Restart gateway
pkill -f "go run ./cmd/gateway"
make run &

echo "Token refreshed and gateway restarted"
```

Run daily via cron:
```bash
0 3 * * * /path/to/refresh-concourse-token.sh
```

### Option 3: Concourse Service Account (Best for Production)

Create a dedicated service account for the gateway.

**Problem**: Concourse local auth doesn't support API-friendly service accounts well. This requires:

1. **Add OAuth Provider** (GitHub, GitLab, OIDC):
```bash
# Example with GitHub OAuth
concourse web \
  --github-client-id YOUR_CLIENT_ID \
  --github-client-secret YOUR_SECRET \
  --main-team-github-user gateway-bot
```

2. **Use Client Credentials Flow**:
- Requires configuring Concourse with an OAuth provider that supports client credentials
- Most production-ready approach

### Option 4: Token Relay via Sidecar

Run a sidecar that keeps a fresh token:

```bash
#!/bin/bash
# scripts/token-sidecar.sh

while true; do
    # Login to refresh token
    fly -t local login -c http://localhost:9001 -u admin -p admin -n

    # Extract and save token
    TOKEN=$(grep -A 2 "local:" ~/.flyrc | grep "value:" | awk '{print $2}')
    echo "$TOKEN" > /tmp/concourse-token

    # Sleep for 12 hours (refresh twice per day)
    sleep 43200
done
```

Gateway reads from `/tmp/concourse-token`:
```go
// Read token from file periodically
func (tm *TokenManager) loadTokenFromFile(path string) {
    ticker := time.NewTicker(1 * time.Minute)
    go func() {
        for range ticker.C {
            data, err := os.ReadFile(path)
            if err == nil {
                tm.mu.Lock()
                tm.token = strings.TrimSpace(string(data))
                tm.tokenExpiry = time.Now().Add(24 * time.Hour)
                tm.mu.Unlock()
            }
        }
    }()
}
```

## Recommended Approach

**For Development/Testing:**
- Use Option 2 (Extract fly token)
- Run refresh script daily

**For Production:**
- Use Option 4 (Token Sidecar) - Simple, reliable, no Concourse reconfiguration needed
- OR Option 3 (Service Account) - If you can configure OAuth providers

## Implementation: Token Sidecar (Recommended)

### Step 1: Create Token Sidecar Script

```bash
#!/bin/bash
# scripts/token-keeper.sh

CONCOURSE_URL="http://localhost:9001"
USERNAME="admin"
PASSWORD="admin"
TOKEN_FILE="/tmp/concourse-gateway-token"

echo "Starting Concourse token keeper..."

while true; do
    echo "[$(date)] Refreshing Concourse token..."

    # Login via fly
    fly -t local login -c "$CONCOURSE_URL" -u "$USERNAME" -p "$PASSWORD" -n

    if [ $? -eq 0 ]; then
        # Extract token from flyrc
        TOKEN=$(grep -A 2 "local:" ~/.flyrc | grep "value:" | awk '{print $2}')

        if [ -n "$TOKEN" ]; then
            echo "$TOKEN" > "$TOKEN_FILE"
            chmod 600 "$TOKEN_FILE"
            echo "[$(date)] Token refreshed successfully"
        else
            echo "[$(date)] ERROR: Failed to extract token"
        fi
    else
        echo "[$(date)] ERROR: Login failed"
    fi

    # Refresh every 12 hours (tokens last ~24 hours)
    sleep 43200
done
```

### Step 2: Run Token Keeper

```bash
chmod +x scripts/token-keeper.sh
./scripts/token-keeper.sh &
```

### Step 3: Update Gateway Config

```yaml
concourse:
  url: "http://localhost:9001"
  team: "main"
  token_file: "/tmp/concourse-gateway-token"  # Read from file
```

### Step 4: Update Token Manager

```go
// Support reading token from file
func (tm *TokenManager) startTokenFileWatcher(path string) {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for {
            select {
            case <-ticker.C:
                data, err := os.ReadFile(path)
                if err == nil {
                    newToken := strings.TrimSpace(string(data))
                    if newToken != "" && newToken != tm.token {
                        tm.mu.Lock()
                        tm.token = newToken
                        tm.tokenExpiry = time.Now().Add(24 * time.Hour)
                        tm.mu.Unlock()
                        log.Println("Token refreshed from file")
                    }
                }
            }
        }
    }()
}
```

## Why Token Sidecar is Best

✅ **No Concourse reconfiguration** - Works with existing setup
✅ **Automatic refresh** - No manual intervention
✅ **Simple** - Just a bash script
✅ **Reliable** - Uses fly CLI (proven to work)
✅ **Secure** - Token file has restricted permissions

## Testing Token Expiration

```bash
# Use an old/invalid token
curl -H "Authorization: Bearer invalid" \
  http://localhost:9001/api/v1/builds/1

# Response: {"error":"unauthorized"}

# Use valid token (from fly)
TOKEN=$(grep -A 2 "local:" ~/.flyrc | grep "value:" | awk '{print $2}')
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:9001/api/v1/builds/1

# Response: Build details (success!)
```

## Summary

| Approach | Complexity | Reliability | Production Ready |
|----------|-----------|-------------|------------------|
| Manual token | Low | Low | ❌ No |
| Daily refresh script | Low | Medium | ⚠️ Dev only |
| Token sidecar | Medium | High | ✅ Yes |
| OAuth service account | High | Highest | ✅ Yes |

**Recommendation**: Start with **Token Sidecar** (Option 4) for a production-ready solution without Concourse reconfiguration.

## Next Steps

1. Implement token sidecar script
2. Update gateway to read from token file
3. Test automatic token refresh
4. Deploy both services together
