# Simple CI Gateway Skill

Use this skill when the user wants to interact with CI/CD pipelines through the Simple CI Gateway.

## What This Tool Does

This project is a stateless CI Gateway that exposes a REST API for managing CI/CD operations:
- Trigger builds for configured jobs
- Monitor build status and progress
- Stream real-time build logs
- Cancel running builds
- Discover teams, pipelines, jobs, and builds

The gateway abstracts Concourse CI behind a provider-agnostic interface.

## Prerequisites

**Before helping the user, verify:**

1. Gateway is running:
   ```bash
   curl http://localhost:8081/health
   ```
   Expected response: `{"status":"ok"}`

2. If not running, start it:
   ```bash
   make run
   ```

3. Verify authentication is configured in `.env`

## Authentication

**All API requests (except /health) require Bearer token authentication.**

```
Authorization: Bearer {API_KEY}
```

**Default API keys** (from `.env`):
- `dev-key-12345` - Local development
- `dashboard-key-67890` - Dashboard access

**Always use the appropriate API key in requests.**

## API Endpoints

### Health Check (No Auth)
```bash
GET http://localhost:8081/health
GET http://localhost:8081/health?detailed=true
```

### List Jobs
```bash
GET http://localhost:8081/v1/jobs
Authorization: Bearer dev-key-12345
```

Returns all configured jobs with their job_id, project, display_name, and provider details.

### Trigger Build
```bash
POST http://localhost:8081/v1/jobs/{job_id}/runs
Authorization: Bearer dev-key-12345
Content-Type: application/json

{
  "parameters": {
    "git_sha": "abc123",
    "environment": "staging"
  },
  "idempotency_key": "optional-unique-key"
}
```

Returns: `run_id` in format `team:pipeline:job:build_id`

### Get Build Status
```bash
GET http://localhost:8081/v1/runs/{run_id}
Authorization: Bearer dev-key-12345
```

**Run ID format:** `team:pipeline:job:build_id`
Example: `main:example-pipeline:hello-job:1049`

**Status values:**
- `queued` - Waiting to start
- `running` - In progress
- `succeeded` - Completed successfully
- `failed` - Build failed
- `canceled` - User canceled
- `errored` - Error occurred
- `unknown` - Status unknown

### Stream Build Logs (SSE)
```bash
GET http://localhost:8081/v1/runs/{run_id}/events
Authorization: Bearer dev-key-12345
```

Returns Server-Sent Events stream with log lines and status updates.

### Cancel Build
```bash
POST http://localhost:8081/v1/runs/{run_id}/cancel
Authorization: Bearer dev-key-12345
```

Returns: 204 No Content

### Discovery API

**List Teams:**
```bash
GET http://localhost:8081/v1/discovery/teams
```

**List Pipelines:**
```bash
GET http://localhost:8081/v1/discovery/pipelines?search=my&paused=false&archived=false
```

**List Jobs:**
```bash
GET http://localhost:8081/v1/discovery/pipelines/{pipeline}/jobs?search=build&paused=false
```

**List Builds:**
```bash
GET http://localhost:8081/v1/discovery/pipelines/{pipeline}/jobs/{job}/builds?limit=10
```

**Get Build Details:**
```bash
GET http://localhost:8081/v1/builds/{build_id}
```

## Common Workflows

### When User Wants to Trigger a Build

1. First, list available jobs to show options:
   ```bash
   curl -H "Authorization: Bearer dev-key-12345" http://localhost:8081/v1/jobs
   ```

2. Ask which job to trigger (or use the job_id if provided)

3. Trigger the build:
   ```bash
   curl -X POST \
     -H "Authorization: Bearer dev-key-12345" \
     -H "Content-Type: application/json" \
     -d '{"parameters": {"git_sha": "abc123"}}' \
     http://localhost:8081/v1/jobs/{job_id}/runs
   ```

4. Extract the `run_id` from response

5. Offer to monitor the build status

### When User Wants to Check Build Status

1. Get the run_id (either from previous trigger or ask user)

2. Check status:
   ```bash
   curl -H "Authorization: Bearer dev-key-12345" \
     http://localhost:8081/v1/runs/{run_id}
   ```

3. Show status, start time, and end time if available

### When User Wants to Monitor Build Progress

1. Offer to stream logs in real-time:
   ```bash
   curl -N -H "Authorization: Bearer dev-key-12345" \
     http://localhost:8081/v1/runs/{run_id}/events
   ```

2. Parse SSE events and display log lines

3. Watch for status changes and notify when complete

### When User Wants to Cancel a Build

1. Confirm they want to cancel

2. Cancel the build:
   ```bash
   curl -X POST \
     -H "Authorization: Bearer dev-key-12345" \
     http://localhost:8081/v1/runs/{run_id}/cancel
   ```

3. Verify cancellation succeeded (204 response)

### When User Wants to Explore Available Pipelines

1. List all pipelines:
   ```bash
   curl -H "Authorization: Bearer dev-key-12345" \
     http://localhost:8081/v1/discovery/pipelines
   ```

2. Filter by search term if needed

3. Show jobs for selected pipeline

4. Display recent builds with status

## Error Handling

**HTTP Status Codes:**
- 200: Success
- 201: Build triggered successfully
- 204: Action completed (cancel)
- 400: Invalid request body
- 401: Invalid or missing API key
- 404: Job or run not found
- 500: Internal server error
- 502: Provider (Concourse) connection issue

**When you encounter errors:**
- 401: Check API key in `.env` file
- 404: Verify job_id exists via `/v1/jobs` endpoint
- 502: Check if Concourse is running and accessible

**Error response format:**
```json
{
  "error": {
    "message": "job not found",
    "code": 404
  }
}
```

## Tips for Helping Users

1. **Always check health first** before attempting operations
2. **List jobs** to help users discover available job_id values
3. **Use descriptive parameter names** when showing examples
4. **Offer to monitor builds** after triggering them
5. **Parse and format** JSON responses for readability
6. **Handle run_id format carefully** - it's colon-separated with 4 parts
7. **Provide context** about build status values
8. **Suggest next actions** based on build results

## Configuration Files

- **`.env`**: Gateway configuration (port, API keys, Concourse connection)
- **`configs/jobs.yaml`**: Job definitions mapping job_id to Concourse pipelines
- **Server Port**: Default 8081 (configurable via SERVER_PORT in .env)
- **Base URL**: http://localhost:8081

## Quick Reference

| Task | Endpoint | Method |
|------|----------|--------|
| Check health | `/health` | GET |
| List jobs | `/v1/jobs` | GET |
| Trigger build | `/v1/jobs/{job_id}/runs` | POST |
| Get status | `/v1/runs/{run_id}` | GET |
| Stream logs | `/v1/runs/{run_id}/events` | GET (SSE) |
| Cancel build | `/v1/runs/{run_id}/cancel` | POST |
| List pipelines | `/v1/discovery/pipelines` | GET |
| List jobs in pipeline | `/v1/discovery/pipelines/{pipeline}/jobs` | GET |
| List builds | `/v1/discovery/pipelines/{pipeline}/jobs/{job}/builds` | GET |
| Get build details | `/v1/builds/{build_id}` | GET |

**Default Auth:** `Authorization: Bearer dev-key-12345`
