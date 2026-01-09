# Quick Start Guide

## Current Status: ✅ Gateway is Running!

The gateway is currently running on **port 8081** with 2 jobs configured from your Concourse pipeline.

## Test It Now

### 1. Health Check (No Auth Required)
```bash
curl http://localhost:8081/health
```
Response: `{"status":"ok"}`

### 2. List Jobs (With API Key)
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs
```

This returns:
- **job_hello** → `example-pipeline/hello-job`
- **job_tests** → `example-pipeline/test-job`

## Add More Jobs from Your Concourse

### Step 1: Find Your Pipelines
```bash
fly -t local pipelines
```

### Step 2: Find Jobs in Each Pipeline
```bash
fly -t local jobs -p YOUR_PIPELINE_NAME
```

### Step 3: Add to configs/jobs.yaml
```yaml
jobs:
  # ... existing jobs ...

  - job_id: "your_new_job"           # Unique ID for API
    project: "your-project"           # Logical grouping
    display_name: "Your Job Name"     # Human-readable
    environment: "prod"               # Tag
    provider:
      kind: "concourse"
      ref:
        team: "main"                  # Your Concourse team
        pipeline: "your-pipeline"     # From step 1
        job: "your-job-name"          # From step 2
```

### Step 4: Restart Gateway
```bash
make run
```

## Example: Complete Workflow

```bash
# 1. Check Concourse for a pipeline
fly -t local pipelines
# Output: my-app-pipeline

# 2. Check jobs in that pipeline
fly -t local jobs -p my-app-pipeline
# Output: build, test, deploy

# 3. Add to configs/jobs.yaml
cat >> configs/jobs.yaml <<'EOF'

  - job_id: "myapp_build"
    project: "my-app"
    display_name: "Build"
    environment: "prod"
    provider:
      kind: "concourse"
      ref:
        team: "main"
        pipeline: "my-app-pipeline"
        job: "build"

  - job_id: "myapp_test"
    project: "my-app"
    display_name: "Test"
    environment: "prod"
    provider:
      kind: "concourse"
      ref:
        team: "main"
        pipeline: "my-app-pipeline"
        job: "test"
EOF

# 4. Restart gateway
pkill -f "go run ./cmd/gateway"
make run

# 5. Verify new jobs appear
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs
```

## API Endpoints Reference

All endpoints require: `Authorization: Bearer dev-key-12345`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth) |
| `/v1/jobs` | GET | List all jobs |
| `/v1/jobs/{job_id}/runs` | POST | Trigger a run |
| `/v1/runs/{run_id}` | GET | Get run status |
| `/v1/runs/{run_id}/events` | GET | Stream logs (SSE) |
| `/v1/runs/{run_id}/cancel` | POST | Cancel run |

## Run ID Format

Format: `team:pipeline:job:build_id`

Example: `main:example-pipeline:hello-job:1049`

## Current Configuration

**Gateway:** `configs/gateway.yaml`
- Port: 8081
- Concourse: http://localhost:9001
- Team: main
- Auth: API keys (dev-key-12345, dashboard-key-67890)

**Jobs:** `configs/jobs.yaml`
- 2 jobs currently configured
- Maps to example-pipeline in Concourse

## Next Steps

1. ✅ Gateway is running and working
2. 📋 Add your Concourse pipelines to `configs/jobs.yaml`
3. 🔐 Configure Concourse service account (see STATUS.md)
4. 🚀 Integrate with your systems using the API

## Get Help

- `README.md` - Full documentation
- `TESTING.md` - Complete testing guide
- `STATUS.md` - Current implementation status
- `DESIGN.md` - Architecture and design decisions

## Troubleshooting

**Port in use?**
```bash
lsof -ti:8081 | xargs kill -9
make run
```

**Jobs not showing?**
- Check `configs/jobs.yaml` syntax
- Verify Concourse pipeline/job names match exactly
- Restart gateway after config changes

**Authentication errors?**
- See STATUS.md for Concourse service account setup
- This is the only remaining item to configure
