# Simple CI Gateway

A stateless, provider-agnostic CI Gateway that exposes a clean REST API backed by Concourse CI.

## Features

- **Stateless Architecture**: No database required, fully in-memory operation
- **Provider Abstraction**: Clean interface for supporting multiple CI systems
- **Simple API**: RESTful endpoints for jobs, runs, status, and logs
- **API Key Authentication**: Secure access control with Bearer tokens
- **SSE Streaming**: Real-time build logs via Server-Sent Events
- **Concourse Integration**: Full support for Concourse CI pipelines

## Architecture

```
┌─────────┐
│ Clients │
└────┬────┘
     │
     v
┌──────────────────┐
│  Generic CI API  │ (Chi Router + Handlers)
└────┬─────────────┘
     │
     v
┌──────────────────┐
│ Service Layer    │ (Business Logic)
└────┬─────────────┘
     │
     v
┌──────────────────┐
│ Provider Adapter │ (Concourse Implementation)
└────┬─────────────┘
     │
     v
┌──────────────────┐
│  Concourse ATC   │
└──────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- Running Concourse CI instance
- Concourse pipelines and jobs configured

### Installation

1. Clone the repository:
```bash
cd simple-ci
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the gateway:

Create a `.env` file from the example:
```bash
cp .env.example .env
```

Edit `.env` to match your Concourse setup:
```bash
# Server
SERVER_PORT=8081

# Authentication
API_KEYS=local-dev:dev-key-12345,ci-dashboard:dashboard-key-67890

# Concourse CI
CONCOURSE_URL=http://localhost:9001
CONCOURSE_TEAM=main
CONCOURSE_USERNAME=admin
CONCOURSE_PASSWORD=admin
CONCOURSE_BEARER_TOKEN=your-token-here

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Jobs Configuration
JOBS_FILE=configs/jobs.yaml
```

Edit `configs/jobs.yaml` to match your Concourse pipelines:
```yaml
jobs:
  - job_id: "job_example_hello"
    project: "example"
    display_name: "Hello World Job"
    environment: "dev"
    provider:
      kind: "concourse"
      ref:
        team: "main"
        pipeline: "example-pipeline"
        job: "hello-job"
```

4. Run the gateway:
```bash
make run
```

The gateway will start on port 8081 (configurable via SERVER_PORT in .env).

## API Endpoints

### Health Check

```bash
GET /health
```

No authentication required.

**Response:**
```json
{
  "status": "ok"
}
```

### List Jobs

```bash
GET /v1/jobs
```

Lists all configured jobs.

**Example:**
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/jobs
```

**Response:**
```json
{
  "jobs": [
    {
      "job_id": "job_example_hello",
      "project": "example",
      "display_name": "Hello World Job",
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

### Trigger a Run

```bash
POST /v1/jobs/{job_id}/runs
```

Triggers a new run for the specified job.

**Request Body:**
```json
{
  "parameters": {
    "git_sha": "abc123",
    "environment": "staging"
  },
  "idempotency_key": "optional-unique-key"
}
```

**Example:**
```bash
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"git_sha": "abc123"}}' \
  http://localhost:8080/v1/jobs/job_example_hello/runs
```

**Response:**
```json
{
  "run": {
    "run_id": "main/example-pipeline/hello-job/123",
    "job_id": "job_example_hello",
    "status": "queued",
    "created_at": "2026-01-08T18:22:11Z",
    "started_at": null,
    "finished_at": null
  }
}
```

### Get Run Status

```bash
GET /v1/runs/{run_id}
```

Retrieves the current status of a run.

**Example:**
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/runs/main/example-pipeline/hello-job/123
```

**Response:**
```json
{
  "run": {
    "run_id": "main/example-pipeline/hello-job/123",
    "status": "running",
    "created_at": "2026-01-08T18:22:11Z",
    "started_at": "2026-01-08T18:22:15Z",
    "finished_at": null
  }
}
```

**Run Status Values:**
- `queued` - Build is queued
- `running` - Build is currently running
- `succeeded` - Build completed successfully
- `failed` - Build failed
- `canceled` - Build was canceled
- `errored` - Build encountered an error
- `unknown` - Status is unknown

### Stream Run Events

```bash
GET /v1/runs/{run_id}/events
```

Streams build events and logs via Server-Sent Events (SSE).

**Example:**
```bash
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/runs/main/example-pipeline/hello-job/123/events
```

**Response** (streaming):
```
data: {"event":"status","data":{"status":"running"}}

data: {"event":"log","data":{"stream":"stdout","line":"Building..."}}

data: {"event":"log","data":{"stream":"stdout","line":"Tests passing..."}}
```

### Cancel Run

```bash
POST /v1/runs/{run_id}/cancel
```

Cancels a running build.

**Example:**
```bash
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/runs/main/example-pipeline/hello-job/123/cancel
```

**Response:**
```
HTTP 204 No Content
```

## Configuration

### Gateway Configuration (`.env`)

Configuration is managed through environment variables. Create a `.env` file from `.env.example`:

```bash
# Simple CI Gateway Configuration

# Server
SERVER_PORT=8081                               # HTTP server port
SERVER_READ_TIMEOUT=30s                        # Read timeout
SERVER_WRITE_TIMEOUT=30s                       # Write timeout

# Authentication
# Comma-separated list of name:key pairs
API_KEYS=local-dev:dev-key-12345,ci-dashboard:dashboard-key-67890

# Concourse CI
CONCOURSE_URL=http://localhost:9001            # Concourse URL
CONCOURSE_TEAM=main                            # Concourse team
CONCOURSE_USERNAME=admin                       # Concourse username
CONCOURSE_PASSWORD=admin                       # Concourse password
CONCOURSE_BEARER_TOKEN=Te/3FtIKdzJpCWmlE9TYQ3QRHxnrtmFpAAAAAA  # Optional: Pre-configured token
CONCOURSE_TOKEN_REFRESH_MARGIN=5m              # Token refresh margin

# Logging
LOG_LEVEL=info                                 # Log level: debug, info, warn, error
LOG_FORMAT=json                                # Log format: json or text

# Jobs Configuration
JOBS_FILE=configs/jobs.yaml                    # Path to jobs definition file
```

### Jobs Configuration (`configs/jobs.yaml`)

```yaml
jobs:
  - job_id: "job_payments_build_test"    # Unique job ID
    project: "payments"                   # Project name
    display_name: "Build & Test"          # Display name
    environment: "prod"                   # Environment
    provider:
      kind: "concourse"                   # Provider type
      ref:
        team: "main"                      # Concourse team
        pipeline: "payments"              # Pipeline name
        job: "build-test"                 # Job name
```

## Development

### Build

```bash
make build
```

Binary will be created at `bin/gateway`.

### Run Locally

```bash
make run
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Lint

```bash
make lint
```

Requires `golangci-lint` to be installed.

## Docker

### Build Docker Image

```bash
make docker-build
```

### Run in Docker

```bash
make docker-run
```

Or with custom configuration:

```bash
docker run -p 8081:8081 \
  --env-file .env \
  -v $(PWD)/configs:/app/configs \
  simple-ci-gateway:latest
```

## Project Structure

```
simple-ci/
├── cmd/
│   └── gateway/
│       └── main.go                    # Application entrypoint
├── internal/
│   ├── api/                           # HTTP API layer
│   │   ├── handlers.go                # Request handlers
│   │   ├── middleware.go              # Auth middleware
│   │   └── routes.go                  # Router setup
│   ├── config/                        # Configuration
│   │   ├── config.go                  # Gateway config
│   │   └── jobs.go                    # Job definitions
│   ├── models/                        # Domain models
│   │   └── models.go                  # Job, Run, Status types
│   ├── provider/                      # Provider abstraction
│   │   ├── provider.go                # Provider interface
│   │   ├── errors.go                  # Error types
│   │   └── concourse/                 # Concourse implementation
│   │       ├── adapter.go             # Provider adapter
│   │       ├── auth.go                # Token manager
│   │       ├── client.go              # ATC API client
│   │       └── mapper.go              # Status mapping
│   └── service/                       # Business logic
│       └── service.go                 # Service layer
├── pkg/
│   └── logger/
│       └── logger.go                  # Structured logging
└── configs/
    └── jobs.yaml                      # Job definitions
├── .env.example                       # Environment configuration template
└── .env                               # Environment configuration (gitignored)
```

## Error Handling

The API returns standard HTTP status codes:

- `200 OK` - Successful request
- `201 Created` - Resource created (trigger run)
- `204 No Content` - Successful operation with no content (cancel)
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Missing or invalid API key
- `404 Not Found` - Job or run not found
- `500 Internal Server Error` - Server error
- `502 Bad Gateway` - Provider error

Error responses include a JSON body:

```json
{
  "error": {
    "message": "job not found",
    "code": 404
  }
}
```

## Security

- **API Keys**: Configure API keys in `.env` file (format: `API_KEYS=name:key,name2:key2`)
- **Production**: Use environment variables for sensitive values (never commit `.env` to version control)
- **HTTPS**: Deploy behind a reverse proxy (nginx, Caddy) with TLS
- **Token Management**: Use CONCOURSE_BEARER_TOKEN for pre-configured tokens or automated refresh scripts

## Future Enhancements

- Dynamic job loading (reload without restart)
- Additional providers (GitHub Actions, Buildkite)
- Multi-tenancy support
- Prometheus metrics
- Webhook notifications
- Artifacts API
- Run history caching

## License

See LICENSE file for details.
