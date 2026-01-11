# Testing Guide - Simple CI Gateway

This guide shows how to configure the gateway with existing Concourse pipelines and test all endpoints.

## Setup

### 1. Check Existing Concourse Pipelines

First, check what pipelines exist in your Concourse instance:

```bash
fly -t local pipelines
```

Example output:
```
name              paused  public  last updated
example-pipeline  no      no      2026-01-08 18:16:52 -0800 PST
```

### 2. List Jobs in Pipeline

Check what jobs are in each pipeline:

```bash
fly -t local jobs -p example-pipeline
```

Example output:
```
name       paused  status     next
hello-job  no      succeeded  n/a
test-job   no      n/a        n/a
```

### 3. Configure Gateway Jobs

Create `configs/jobs.yaml` with your existing Concourse jobs:

```yaml
jobs:
  # Map each Concourse job to a gateway job
  - job_id: "job_hello"              # Unique ID for API
    project: "example"                # Logical grouping
    display_name: "Hello Job"         # Human-readable name
    environment: "dev"                # Environment tag
    provider:
      kind: "concourse"
      ref:
        team: "main"                  # Concourse team name
        pipeline: "example-pipeline"  # Pipeline name from fly
        job: "hello-job"              # Job name from fly

  - job_id: "job_tests"
    project: "example"
    display_name: "Test Job"
    environment: "dev"
    provider:
      kind: "concourse"
      ref:
        team: "main"
        pipeline: "example-pipeline"
        job: "test-job"
```

### 4. Configure Gateway Connection

Edit `configs/gateway.yaml`:

```yaml
concourse:
  url: "http://localhost:9001"  # Your Concourse URL
  team: "main"                   # Default team
  username: "admin"              # Concourse username
  password: "admin"              # Concourse password
```

### 5. Start the Gateway

```bash
make run
```

The gateway runs on port **8081** (not 9001 - that's where Concourse runs).

## Testing the API

### Test 1: Health Check (No Auth)

```bash
curl http://localhost:8081/health
```

Response:
```json
{"status": "ok"}
```

### Test 2: List Jobs

```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs | python3 -m json.tool
```

Response:
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
                    "job": "hello-job",
                    "pipeline": "example-pipeline",
                    "team": "main"
                }
            }
        }
    ]
}
```

### Test 3: Trigger a Build (Using fly CLI)

For now, trigger builds using the fly CLI:

```bash
fly -t local trigger-job -j example-pipeline/hello-job --watch
```

This creates a build with an ID (e.g., build #2).

### Test 4: Get Run Status

Use the run_id format: `team:pipeline:job:build_id`

```bash
# For build #2 of hello-job
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:2 \
  | python3 -m json.tool
```

Expected Response:
```json
{
    "run": {
        "run_id": "main:example-pipeline:hello-job:2",
        "status": "succeeded",
        "created_at": "2026-01-08T18:25:00Z",
        "started_at": "2026-01-08T18:25:05Z",
        "finished_at": "2026-01-08T18:25:15Z"
    }
}
```

### Test 5: Stream Build Events

```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:2/events
```

This streams Server-Sent Events with build logs.

### Test 6: Cancel a Running Build

```bash
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:example-pipeline:hello-job:3/cancel
```

Response: HTTP 204 No Content

## API Authentication

The gateway uses API key authentication. Keys are configured in `configs/gateway.yaml`:

```yaml
auth:
  api_keys:
    - name: "local-dev"
      key: "dev-key-12345"
    - name: "ci-dashboard"
      key: "dashboard-key-67890"
```

Use the key in the Authorization header:
```
Authorization: Bearer dev-key-12345
```

## Run ID Format

Run IDs use the format: `team:pipeline:job:build_id`

Examples:
- `main:example-pipeline:hello-job:1`
- `main:payments-pipeline:build-test:42`
- `prod-team:api-pipeline:deploy:123`

The `:` separator is URL-safe (unlike `/`).

## How to Preload Data from Concourse

### Method 1: Manual Configuration

1. List all pipelines: `fly -t local pipelines`
2. For each pipeline, list jobs: `fly -t local jobs -p PIPELINE_NAME`
3. Add each job to `configs/jobs.yaml`
4. Restart the gateway

### Method 2: Script to Generate Config

Create a script to auto-generate the config:

```bash
#!/bin/bash
# generate-jobs-config.sh

TEAM="main"
OUTPUT="configs/jobs-generated.yaml"

echo "jobs:" > $OUTPUT

fly -t local pipelines | tail -n +1 | while read -r line; do
  PIPELINE=$(echo "$line" | awk '{print $1}')

  fly -t local jobs -p "$PIPELINE" | tail -n +1 | while read -r job_line; do
    JOB=$(echo "$job_line" | awk '{print $1}')

    cat >> $OUTPUT <<EOF
  - job_id: "job_${PIPELINE}_${JOB}"
    project: "$PIPELINE"
    display_name: "$JOB"
    environment: "prod"
    provider:
      kind: "concourse"
      ref:
        team: "$TEAM"
        pipeline: "$PIPELINE"
        job: "$JOB"

EOF
  done
done

echo "Generated: $OUTPUT"
```

Run it:
```bash
chmod +x generate-jobs-config.sh
./generate-jobs-config.sh
mv configs/jobs-generated.yaml configs/jobs.yaml
make run
```

## Troubleshooting

### Port Already in Use

If you see "bind: address already in use":

```bash
# Find and kill the process
lsof -ti:8081 | xargs kill -9

# Or change the port in configs/gateway.yaml
server:
  port: 8082  # Use a different port
```

### Authentication Errors

Check Concourse credentials:
```bash
fly -t local login -c http://localhost:9001 -u admin -p admin
```

### Run Not Found

Make sure:
1. The build exists in Concourse: `fly -t local builds`
2. The run_id format is correct: `team:pipeline:job:build_id`
3. The build number matches exactly

## Complete Example Workflow

```bash
# 1. Setup Concourse pipeline
fly -t local set-pipeline -p my-pipeline -c pipeline.yml
fly -t local unpause-pipeline -p my-pipeline

# 2. Configure gateway
cat > configs/jobs.yaml <<EOF
jobs:
  - job_id: "my_app_build"
    project: "my-app"
    display_name: "Build"
    environment: "prod"
    provider:
      kind: "concourse"
      ref:
        team: "main"
        pipeline: "my-pipeline"
        job: "build"
EOF

# 3. Start gateway
make run

# 4. List jobs via API
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/jobs

# 5. Trigger build (using fly for now)
fly -t local trigger-job -j my-pipeline/build

# 6. Check status via API
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8081/v1/runs/main:my-pipeline:build:1
```

## Next Steps

- Configure proper Concourse service accounts for triggering builds
- Add more pipelines to jobs.yaml
- Integrate with your CI/CD systems
- Deploy the gateway to a server
- Use HTTPS with reverse proxy (nginx, Caddy)
