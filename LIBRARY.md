# Using Simple CI Gateway as a Library

The Simple CI Gateway can be embedded into your Go applications as a library, giving you flexible options for CI integration.

## Installation

Add the library to your Go project:

```bash
go get github.com/lei/simple-ci
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/lei/simple-ci/internal/models"
    "github.com/lei/simple-ci/pkg/gateway"
)

func main() {
    // Define jobs
    jobs := []*models.Job{
        {
            JobID:       "my-build-job",
            Project:     "myapp",
            DisplayName: "Build & Test",
            Environment: "production",
            Provider: models.JobProviderConfig{
                Kind: "concourse",
                Ref: map[string]interface{}{
                    "team":     "main",
                    "pipeline": "myapp-pipeline",
                    "job":      "build-test",
                },
            },
        },
    }

    // Configure the gateway
    cfg := &gateway.Config{
        Server: gateway.ServerConfig{
            Port:         8080,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
        },
        Auth: gateway.AuthConfig{
            APIKeys: []gateway.APIKey{
                {Name: "my-app", Key: "your-secret-key"},
            },
        },
        Provider: gateway.ProviderConfig{
            Kind: "concourse",
            Concourse: &gateway.ConcourseConfig{
                URL:                "http://localhost:9001",
                Team:               "main",
                Username:           "admin",
                Password:           "admin",
                TokenRefreshMargin: 5 * time.Minute,
            },
        },
        Jobs: jobs,
        Logging: gateway.LoggingConfig{
            Level:  "info",
            Format: "json",
        },
    }

    // Create and start the gateway
    gw, err := gateway.New(cfg)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    if err := gw.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Integration Patterns

### 1. Standalone Gateway

Run the gateway as a dedicated service (same as the CLI version):

```go
gw, err := gateway.New(cfg)
if err != nil {
    log.Fatal(err)
}

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

// This blocks until shutdown
if err := gw.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 2. Embedded in Existing HTTP Server

Mount the gateway under a specific path in your application:

```go
gw, err := gateway.New(cfg)
if err != nil {
    log.Fatal(err)
}

// Create your application's router
mux := http.NewServeMux()

// Mount CI Gateway under /ci
mux.Handle("/ci/", http.StripPrefix("/ci", gw.Handler()))

// Add your application's routes
mux.HandleFunc("/", myHomeHandler)
mux.HandleFunc("/api/v1/users", myUsersHandler)

// Start server
http.ListenAndServe(":8080", mux)
```

Now the CI Gateway is available at:
- `http://localhost:8080/ci/health`
- `http://localhost:8080/ci/v1/jobs`
- etc.

### 3. Programmatic Access (No HTTP Server)

Use the service layer directly for programmatic CI control:

```go
gw, err := gateway.New(cfg)
if err != nil {
    log.Fatal(err)
}

svc := gw.Service()
ctx := context.Background()

// List jobs
jobs := svc.ListJobs(ctx)
for _, job := range jobs {
    fmt.Printf("Job: %s\n", job.DisplayName)
}

// Trigger a run
run, err := svc.TriggerRun(ctx, "my-build-job", map[string]interface{}{
    "git_sha": "abc123",
}, "")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Triggered run: %s\n", run.RunID)

// Check status
status, err := svc.GetRun(ctx, run.RunID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status)
```

Perfect for:
- CLI tools
- Scheduled tasks
- Integration tests
- Custom automation

### 4. Environment-Based Configuration

Use the same configuration approach as the standalone gateway:

```go
// Requires .env file and jobs.yaml
gw, err := gateway.NewFromEnv("configs/jobs.yaml")
if err != nil {
    log.Fatal(err)
}

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

if err := gw.Start(ctx); err != nil {
    log.Fatal(err)
}
```

This reads configuration from environment variables:
- `SERVER_PORT`
- `API_KEYS`
- `CONCOURSE_URL`
- `CONCOURSE_TEAM`
- `CONCOURSE_USERNAME`
- `CONCOURSE_PASSWORD`
- etc.

## Configuration Reference

### Gateway Config

```go
type Config struct {
    Server   ServerConfig
    Auth     AuthConfig
    Provider ProviderConfig
    Jobs     []*models.Job
    Logging  LoggingConfig
}
```

### Server Config

```go
type ServerConfig struct {
    Port         int           // HTTP port (default: 8080)
    ReadTimeout  time.Duration // Request read timeout
    WriteTimeout time.Duration // Response write timeout
}
```

### Auth Config

```go
type AuthConfig struct {
    APIKeys []APIKey // List of API keys
}

type APIKey struct {
    Name string // Key identifier (for logging)
    Key  string // The actual API key
}
```

### Provider Config

```go
type ProviderConfig struct {
    Kind      string            // "concourse" (more providers planned)
    Concourse *ConcourseConfig  // Required when Kind is "concourse"
}

type ConcourseConfig struct {
    URL                string        // Concourse ATC URL
    Team               string        // Team name
    Username           string        // Username (if not using BearerToken)
    Password           string        // Password (if not using BearerToken)
    BearerToken        string        // Pre-configured token (optional)
    TokenRefreshMargin time.Duration // Token refresh margin
}
```

### Job Definition

```go
type Job struct {
    JobID       string            // Unique job identifier
    Project     string            // Project name
    DisplayName string            // Human-readable name
    Environment string            // Environment (dev, staging, prod)
    Provider    JobProviderConfig // Provider-specific config
}

type JobProviderConfig struct {
    Kind string                 // Provider type ("concourse")
    Ref  map[string]interface{} // Provider-specific reference
}
```

For Concourse jobs, the `Ref` map should contain:
```go
Ref: map[string]interface{}{
    "team":     "main",
    "pipeline": "my-pipeline",
    "job":      "my-job",
}
```

### Logging Config

```go
type LoggingConfig struct {
    Level  string // "debug", "info", "warn", "error"
    Format string // "json" or "text"
}
```

## Service Layer API

When using programmatic access, the service layer provides these methods:

```go
// List all configured jobs
func (s *Service) ListJobs(ctx context.Context) []*models.Job

// Trigger a new run
func (s *Service) TriggerRun(ctx context.Context, jobID string,
    params map[string]interface{}, idempotencyKey string) (*models.Run, error)

// Get run status
func (s *Service) GetRun(ctx context.Context, runID string) (*models.Run, error)

// Stream run events (SSE)
func (s *Service) StreamRunEvents(ctx context.Context, runID string,
    writer io.Writer) error

// Cancel a running build
func (s *Service) CancelRun(ctx context.Context, runID string) error

// Health check
func (s *Service) HealthCheck(ctx context.Context) map[string]interface{}
```

For Concourse-specific discovery:

```go
// List teams
func (s *Service) ListTeams(ctx context.Context) ([]concourse.Team, error)

// List pipelines
func (s *Service) ListPipelines(ctx context.Context) ([]concourse.Pipeline, error)

// List jobs in a pipeline
func (s *Service) ListPipelineJobs(ctx context.Context, pipeline string)
    ([]concourse.Job, error)

// List builds for a job
func (s *Service) ListJobBuilds(ctx context.Context, pipeline, job string,
    limit int) ([]concourse.Build, error)

// Get build details
func (s *Service) GetBuildDetails(ctx context.Context, buildID int)
    (*concourse.Build, map[string]interface{}, error)
```

## Examples

See the [examples](./examples) directory for complete working examples:

- **[basic](./examples/basic)** - Standalone gateway with programmatic configuration
- **[embedded](./examples/embedded)** - Gateway embedded in existing HTTP server
- **[programmatic](./examples/programmatic)** - Direct service layer access
- **[env-based](./examples/env-based)** - Environment variable configuration

## Migration from Standalone Gateway

If you're currently using the standalone `cmd/gateway` application and want to embed it:

1. **Keep using environment variables:**
   ```go
   gw, err := gateway.NewFromEnv("configs/jobs.yaml")
   ```

2. **Or switch to programmatic config:**
   ```go
   cfg := &gateway.Config{
       // ... configuration
   }
   gw, err := gateway.New(cfg)
   ```

3. **Update your main.go:**
   ```go
   import "github.com/lei/simple-ci/pkg/gateway"

   func main() {
       gw, err := gateway.NewFromEnv("configs/jobs.yaml")
       if err != nil {
           log.Fatal(err)
       }

       ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
       defer cancel()

       if err := gw.Start(ctx); err != nil {
           log.Fatal(err)
       }
   }
   ```

## Use Cases

### 1. Internal Developer Platform

Embed the gateway in your platform's backend:

```go
// Platform API
mux.HandleFunc("/api/v1/apps", handleApps)
mux.HandleFunc("/api/v1/deployments", handleDeployments)

// CI Gateway
mux.Handle("/api/ci/", http.StripPrefix("/api/ci", gateway.Handler()))
```

### 2. ChatOps Bot

Trigger CI jobs from Slack/Discord:

```go
svc := gateway.Service()

// In your bot's command handler
run, err := svc.TriggerRun(ctx, jobID, params, "")
if err != nil {
    return err
}

bot.Reply(fmt.Sprintf("Build started: %s", run.RunID))
```

### 3. Automated Testing

Trigger builds in integration tests:

```go
func TestDeployment(t *testing.T) {
    svc := testGateway.Service()

    run, err := svc.TriggerRun(ctx, "deploy-staging", map[string]interface{}{
        "version": "v1.2.3",
    }, "")
    require.NoError(t, err)

    // Wait for completion and verify
    // ...
}
```

### 4. Custom CLI Tool

Build a custom CLI for your team:

```go
cmd.Flags().StringVar(&jobID, "job", "", "Job to trigger")
// ...

svc := gateway.Service()
run, err := svc.TriggerRun(ctx, jobID, params, "")
// ...
```

## API Documentation

When running as an HTTP server, the gateway exposes these endpoints:

- `GET /health` - Health check
- `GET /v1/jobs` - List jobs
- `POST /v1/jobs/{job_id}/runs` - Trigger run
- `GET /v1/runs/{run_id}` - Get run status
- `GET /v1/runs/{run_id}/events` - Stream events (SSE)
- `POST /v1/runs/{run_id}/cancel` - Cancel run
- `GET /v1/discovery/teams` - List teams
- `GET /v1/discovery/pipelines` - List pipelines
- And more...

See the [main README](./README.md) for full API documentation.

## License

See [LICENSE](./LICENSE) file for details.
