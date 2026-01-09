# Skills Guide - Using the Generic CI Gateway

## Overview

This guide explains how AI agents and automation tools can interact with the Generic CI Gateway to manage CI/CD pipelines programmatically.

**What this tool does:**
- Provides a unified REST API for CI/CD operations
- Abstracts Concourse CI behind a provider-agnostic interface
- Enables triggering, monitoring, and managing CI builds
- Offers real-time build log streaming

**Gateway Location:** http://localhost:8081 (configurable)

---

## Prerequisites

**Before using the gateway:**
1. Gateway must be running: `make run`
2. You need an API key (from `configs/gateway.yaml`)
3. Jobs must be configured in `configs/jobs.yaml`

**Check gateway status:**
```bash
curl http://localhost:8081/health
# Expected: {"status":"ok"}
```

---

## Authentication

**All API requests (except /health) require authentication:**

```bash
Authorization: Bearer YOUR_API_KEY
```

**Available API keys** (from config):
- `dev-key-12345` - Local development key
- `dashboard-key-67890` - Dashboard key

**Example:**
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs
```

---

## API Endpoints

### 1. Health Check

**Purpose:** Verify gateway is running

```bash
GET /health
```

**No authentication required**

**Response:**
```json
{"status": "ok"}
```

**When to use:**
- Before making other API calls
- Health monitoring
- Readiness checks

---

### 2. List Jobs

**Purpose:** Get all configured CI jobs

```bash
GET /v1/jobs
```

**Headers:**
```
Authorization: Bearer dev-key-12345
```

**Response:**
```json
{
  "jobs": [
    {
      "job_id": "job_hello",
      "project": "example",
      "display_name": "Hello Job",
      "environment": "dev",
      "provider": {
        "kind": "concourse",
        "ref": {
          "team": "main",
          "pipeline": "example-pipeline",
          "job": "hello-job"
        }
      }
    }
  ]
}
```

**When to use:**
- Discovering available jobs
- Building job selection UIs
- Validating job_id exists

**Example usage:**
```bash
# Get all jobs
curl -s -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs | jq '.jobs[].job_id'
```

---

### 3. Trigger a Build

**Purpose:** Start a new CI build

```bash
POST /v1/jobs/{job_id}/runs
```

**Headers:**
```
Authorization: Bearer dev-key-12345
Content-Type: application/json
```

**Request Body:**
```json
{
  "parameters": {
    "git_sha": "abc123",
    "environment": "staging",
    "version": "1.2.3"
  },
  "idempotency_key": "optional-unique-key"
}
```

**Parameters:**
- `parameters` (object, optional): Key-value pairs passed to CI job
- `idempotency_key` (string, optional): Prevent duplicate triggers

**Response (201 Created):**
```json
{
  "run": {
    "run_id": "main:example-pipeline:hello-job:1050",
    "job_id": "job_hello",
    "status": "queued",
    "created_at": "2026-01-08T18:30:00Z",
    "started_at": null,
    "finished_at": null
  }
}
```

**Example:**
```bash
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"git_sha": "abc123"}}' \
  http://localhost:8081/v1/jobs/job_hello/runs
```

**Note:** Currently requires fly CLI for triggering. See AUTHENTICATION.md for details.

---

### 4. Get Build Status

**Purpose:** Check the status of a specific build

```bash
GET /v1/runs/{run_id}
```

**Headers:**
```
Authorization: Bearer dev-key-12345
```

**Run ID Format:** `team:pipeline:job:build_id`
- Example: `main:example-pipeline:hello-job:1049`

**Response:**
```json
{
  "run": {
    "run_id": "main:example-pipeline:hello-job:1049",
    "status": "succeeded",
    "created_at": "2026-01-08T18:20:00Z",
    "started_at": "2026-01-08T18:20:19Z",
    "finished_at": "2026-01-08T18:20:41Z"
  }
}
```

**Status Values:**
- `queued` - Build is waiting to start
- `running` - Build is in progress
- `succeeded` - Build completed successfully
- `failed` - Build failed
- `canceled` - Build was canceled
- `errored` - Build encountered an error
- `unknown` - Status cannot be determined

**Example:**
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:1049
```

**When to use:**
- Polling build status
- Checking if build is complete
- Getting build timing information

---

### 5. Stream Build Logs

**Purpose:** Get real-time build logs via Server-Sent Events (SSE)

```bash
GET /v1/runs/{run_id}/events
```

**Headers:**
```
Authorization: Bearer dev-key-12345
```

**Response:** Server-Sent Events stream

```
data: {"event":"status","data":{"status":"running"}}

data: {"event":"log","data":{"stream":"stdout","line":"Building..."}}

data: {"event":"log","data":{"stream":"stdout","line":"Tests passing..."}}
```

**Event Types:**
- `status` - Build status changed
- `log` - Log line output
- `error` - Error occurred

**Example:**
```bash
curl -N -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:1049/events
```

**When to use:**
- Real-time log monitoring
- Live build feedback
- Debugging build issues

**Note:** Connection stays open until build completes or client disconnects

---

### 6. Cancel Build

**Purpose:** Stop a running build

```bash
POST /v1/runs/{run_id}/cancel
```

**Headers:**
```
Authorization: Bearer dev-key-12345
```

**Response (204 No Content):**
No response body

**Example:**
```bash
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:1049/cancel
```

**When to use:**
- User-initiated cancellation
- Build timeout handling
- Resource cleanup

---

## Common Workflows

### Workflow 1: List and Select Job

```bash
# Step 1: List all jobs
JOBS=$(curl -s -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs)

# Step 2: Extract job IDs
echo "$JOBS" | jq -r '.jobs[].job_id'

# Step 3: Select a job (example: job_hello)
JOB_ID="job_hello"
```

### Workflow 2: Trigger and Monitor Build

```bash
# Step 1: Trigger build
RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"version": "1.0.0"}}' \
  http://localhost:8081/v1/jobs/job_hello/runs)

# Step 2: Extract run_id
RUN_ID=$(echo "$RESPONSE" | jq -r '.run.run_id')
echo "Started build: $RUN_ID"

# Step 3: Poll status until complete
while true; do
  STATUS=$(curl -s -H "Authorization: Bearer dev-key-12345" \
    http://localhost:8081/v1/runs/$RUN_ID | jq -r '.run.status')

  echo "Status: $STATUS"

  if [[ "$STATUS" =~ ^(succeeded|failed|canceled|errored)$ ]]; then
    break
  fi

  sleep 5
done

echo "Build finished with status: $STATUS"
```

### Workflow 3: Stream Logs with Timeout

```bash
# Stream logs with 60-second timeout
timeout 60 curl -N -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:1049/events \
  | while IFS= read -r line; do
    echo "[$(date)] $line"
  done
```

### Workflow 4: Conditional Build Trigger

```bash
# Trigger build only if previous build succeeded
PREV_RUN_ID="main:example-pipeline:hello-job:1048"

STATUS=$(curl -s -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/$PREV_RUN_ID | jq -r '.run.status')

if [ "$STATUS" = "succeeded" ]; then
  echo "Previous build succeeded, triggering new build..."
  curl -X POST \
    -H "Authorization: Bearer dev-key-12345" \
    -H "Content-Type: application/json" \
    -d '{}' \
    http://localhost:8081/v1/jobs/job_hello/runs
else
  echo "Previous build did not succeed: $STATUS"
  exit 1
fi
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Process response |
| 201 | Created | Build triggered successfully |
| 204 | No Content | Action completed (cancel) |
| 400 | Bad Request | Check request body format |
| 401 | Unauthorized | Verify API key |
| 404 | Not Found | Check job_id or run_id |
| 500 | Server Error | Check gateway logs |
| 502 | Bad Gateway | Concourse connection issue |

### Error Response Format

```json
{
  "error": {
    "message": "job not found",
    "code": 404
  }
}
```

### Handling Errors

```bash
# Check response status
HTTP_CODE=$(curl -s -o response.json -w "%{http_code}" \
  -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs)

if [ "$HTTP_CODE" -eq 200 ]; then
  cat response.json | jq .
elif [ "$HTTP_CODE" -eq 401 ]; then
  echo "Authentication failed - check API key"
elif [ "$HTTP_CODE" -eq 404 ]; then
  echo "Resource not found"
else
  echo "Error: HTTP $HTTP_CODE"
  cat response.json | jq -r '.error.message'
fi
```

---

## Agent Integration Patterns

### Pattern 1: Python Agent

```python
import requests
import time

class CIGatewayClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {"Authorization": f"Bearer {api_key}"}

    def list_jobs(self):
        response = requests.get(
            f"{self.base_url}/v1/jobs",
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()["jobs"]

    def get_run_status(self, run_id):
        response = requests.get(
            f"{self.base_url}/v1/runs/{run_id}",
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()["run"]

    def wait_for_completion(self, run_id, poll_interval=5):
        """Wait for build to complete"""
        while True:
            run = self.get_run_status(run_id)
            status = run["status"]

            if status in ["succeeded", "failed", "canceled", "errored"]:
                return run

            time.sleep(poll_interval)

# Usage
client = CIGatewayClient("http://localhost:8081", "dev-key-12345")

# List jobs
jobs = client.list_jobs()
print(f"Found {len(jobs)} jobs")

# Monitor build
run = client.wait_for_completion("main:example-pipeline:hello-job:1049")
print(f"Build finished: {run['status']}")
```

### Pattern 2: Node.js Agent

```javascript
const axios = require('axios');

class CIGatewayClient {
    constructor(baseURL, apiKey) {
        this.client = axios.create({
            baseURL: baseURL,
            headers: { 'Authorization': `Bearer ${apiKey}` }
        });
    }

    async listJobs() {
        const response = await this.client.get('/v1/jobs');
        return response.data.jobs;
    }

    async getRunStatus(runId) {
        const response = await this.client.get(`/v1/runs/${runId}`);
        return response.data.run;
    }

    async waitForCompletion(runId, pollInterval = 5000) {
        while (true) {
            const run = await this.getRunStatus(runId);

            if (['succeeded', 'failed', 'canceled', 'errored'].includes(run.status)) {
                return run;
            }

            await new Promise(resolve => setTimeout(resolve, pollInterval));
        }
    }
}

// Usage
const client = new CIGatewayClient('http://localhost:8081', 'dev-key-12345');

(async () => {
    const jobs = await client.listJobs();
    console.log(`Found ${jobs.length} jobs`);

    const run = await client.waitForCompletion('main:example-pipeline:hello-job:1049');
    console.log(`Build finished: ${run.status}`);
})();
```

### Pattern 3: Bash Script Agent

```bash
#!/bin/bash
# ci-agent.sh - Simple CI Gateway agent

BASE_URL="http://localhost:8081"
API_KEY="dev-key-12345"

# Helper function for API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3

    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Authorization: Bearer $API_KEY" \
            "$BASE_URL$endpoint"
    else
        curl -s -X "$method" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint"
    fi
}

# List jobs
list_jobs() {
    api_call GET "/v1/jobs" | jq -r '.jobs[].job_id'
}

# Get run status
get_status() {
    local run_id=$1
    api_call GET "/v1/runs/$run_id" | jq -r '.run.status'
}

# Wait for completion
wait_for_completion() {
    local run_id=$1

    while true; do
        status=$(get_status "$run_id")
        echo "Status: $status"

        case $status in
            succeeded|failed|canceled|errored)
                echo "Build finished: $status"
                return 0
                ;;
        esac

        sleep 5
    done
}

# Main
echo "Available jobs:"
list_jobs

echo ""
echo "Monitoring build: $1"
wait_for_completion "$1"
```

---

## Best Practices for Agents

### 1. Health Checks

Always verify gateway is available before operations:

```bash
if curl -sf http://localhost:8081/health > /dev/null; then
    echo "Gateway is healthy"
else
    echo "Gateway is not responding"
    exit 1
fi
```

### 2. Retry Logic

Implement retries for transient failures:

```bash
retry_api_call() {
    local max_attempts=3
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if response=$(curl -sf -H "Authorization: Bearer dev-key-12345" \
            http://localhost:8081/v1/jobs); then
            echo "$response"
            return 0
        fi

        echo "Attempt $attempt failed, retrying..."
        attempt=$((attempt + 1))
        sleep 2
    done

    return 1
}
```

### 3. Timeout Handling

Set reasonable timeouts:

```bash
# 30-second timeout for API calls
curl --max-time 30 -H "Authorization: Bearer dev-key-12345" \
    http://localhost:8081/v1/jobs
```

### 4. Logging

Log all interactions for debugging:

```bash
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

log "Fetching jobs..."
JOBS=$(curl -s -H "Authorization: Bearer dev-key-12345" \
    http://localhost:8081/v1/jobs)
log "Found $(echo "$JOBS" | jq '.jobs | length') jobs"
```

### 5. Error Recovery

Handle errors gracefully:

```bash
trigger_build() {
    local job_id=$1

    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Authorization: Bearer dev-key-12345" \
        -H "Content-Type: application/json" \
        -d '{}' \
        "http://localhost:8081/v1/jobs/$job_id/runs")

    http_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)

    case $http_code in
        201)
            echo "Build triggered successfully"
            echo "$body" | jq -r '.run.run_id'
            return 0
            ;;
        404)
            echo "Error: Job not found: $job_id"
            return 1
            ;;
        401)
            echo "Error: Authentication failed"
            return 1
            ;;
        *)
            echo "Error: Unexpected status $http_code"
            echo "$body"
            return 1
            ;;
    esac
}
```

---

## Configuration Discovery

### Get Current Configuration

```bash
# Get available jobs
curl -s -H "Authorization: Bearer dev-key-12345" \
    http://localhost:8081/v1/jobs | jq -r '.jobs[] |
    "ID: \(.job_id), Project: \(.project), Job: \(.provider.ref.job)"'
```

### Validate Job Exists

```bash
check_job_exists() {
    local job_id=$1

    jobs=$(curl -s -H "Authorization: Bearer dev-key-12345" \
        http://localhost:8081/v1/jobs)

    if echo "$jobs" | jq -e ".jobs[] | select(.job_id == \"$job_id\")" > /dev/null; then
        return 0
    else
        return 1
    fi
}

if check_job_exists "job_hello"; then
    echo "Job exists"
else
    echo "Job not found"
fi
```

---

## Troubleshooting for Agents

### Connection Issues

```bash
# Test basic connectivity
if ! curl -sf http://localhost:8081/health > /dev/null; then
    echo "Cannot connect to gateway at http://localhost:8081"
    echo "Possible causes:"
    echo "  - Gateway is not running (run: make run)"
    echo "  - Wrong port (check configs/gateway.yaml)"
    echo "  - Network issue"
    exit 1
fi
```

### Authentication Issues

```bash
# Verify API key works
response=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer dev-key-12345" \
    http://localhost:8081/v1/jobs)

http_code=$(echo "$response" | tail -1)

if [ "$http_code" = "401" ]; then
    echo "Authentication failed"
    echo "Check API key in configs/gateway.yaml"
    exit 1
fi
```

### Invalid Run ID

```bash
# Validate run_id format
validate_run_id() {
    local run_id=$1

    # Expected format: team:pipeline:job:build_id
    if [[ $run_id =~ ^[^:]+:[^:]+:[^:]+:[0-9]+$ ]]; then
        return 0
    else
        echo "Invalid run_id format: $run_id"
        echo "Expected: team:pipeline:job:build_id"
        echo "Example: main:example-pipeline:hello-job:1049"
        return 1
    fi
}
```

---

## Summary

**Key Points for Agents:**

1. **Always authenticate** with Bearer token (except /health)
2. **Check gateway health** before operations
3. **Use correct run_id format**: `team:pipeline:job:build_id`
4. **Poll status** for build completion (5-10 second intervals)
5. **Handle errors** gracefully with retries
6. **Set timeouts** for all HTTP requests
7. **Log operations** for debugging

**Quick Reference:**

| Task | Endpoint | Method |
|------|----------|--------|
| Health check | `/health` | GET |
| List jobs | `/v1/jobs` | GET |
| Trigger build | `/v1/jobs/{job_id}/runs` | POST |
| Get status | `/v1/runs/{run_id}` | GET |
| Stream logs | `/v1/runs/{run_id}/events` | GET |
| Cancel build | `/v1/runs/{run_id}/cancel` | POST |

**Gateway:** http://localhost:8081
**Auth:** `Authorization: Bearer dev-key-12345`

For more details, see README.md, TESTING.md, and FINAL-SUMMARY.md.
