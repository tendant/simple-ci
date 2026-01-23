# Simple CI Gateway - Examples

This directory contains examples showing different ways to use the Simple CI Gateway as a library in your Go applications.

## Running the Examples

All examples require a running Concourse CI instance. Update the configuration in each example to match your Concourse setup.

### Basic Standalone Gateway

The simplest way to use the library - create and start a gateway programmatically.

```bash
cd examples/basic
go run main.go
```

This example shows:
- Creating jobs programmatically
- Configuring the gateway with code
- Starting the gateway server
- Graceful shutdown

### Embedded Gateway

Integrate the CI Gateway into an existing HTTP application.

```bash
cd examples/embedded
go run main.go
```

This example shows:
- Mounting the gateway under a custom path (`/ci/`)
- Adding custom application routes alongside the gateway
- Using a single HTTP server for both

### Programmatic Access

Use the service layer directly without running an HTTP server.

```bash
cd examples/programmatic
go run main.go
```

This example shows:
- Direct access to the service layer
- Triggering jobs programmatically
- Polling run status
- Perfect for CLI tools or scheduled tasks

### Environment-Based Configuration

Use environment variables and config files (same as standalone gateway).

```bash
cd examples/env-based
cp ../../.env.example .env
# Edit .env to match your setup
go run main.go
```

This example shows:
- Loading configuration from `.env` file
- Using `gateway.NewFromEnv()` for easy migration from standalone gateway
- Environment variable-based configuration

## Example Configurations

### Job Definition

```go
job := &models.Job{
    JobID:       "job_example_hello",
    Project:     "example",
    DisplayName: "Hello World Job",
    Environment: "dev",
    Provider: models.JobProviderConfig{
        Kind: "concourse",
        Ref: map[string]interface{}{
            "team":     "main",
            "pipeline": "example-pipeline",
            "job":      "hello-job",
        },
    },
}
```

### Gateway Configuration

```go
cfg := &gateway.Config{
    Server: gateway.ServerConfig{
        Port:         8080,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    },
    Auth: gateway.AuthConfig{
        APIKeys: []gateway.APIKey{
            {Name: "my-app", Key: "secret-key-here"},
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
```

## Testing the Gateway

Once an example is running, test it with curl:

```bash
# Health check (no auth required)
curl http://localhost:8080/health

# List jobs
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/jobs

# Trigger a run
curl -X POST \
  -H "Authorization: Bearer dev-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"git_sha": "abc123"}}' \
  http://localhost:8080/v1/jobs/job_example_hello/runs
```

For embedded example, use `/ci/` prefix:
```bash
curl http://localhost:8080/ci/health
```

## Learn More

- See [main README](../README.md) for full API documentation
- Read [pkg/gateway/doc.go](../pkg/gateway/doc.go) for package documentation
- Explore the [internal](../internal) packages for implementation details
