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
	// Define jobs programmatically
	jobs := []*models.Job{
		{
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
		},
	}

	// Create gateway configuration
	cfg := &gateway.Config{
		Server: gateway.ServerConfig{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Auth: gateway.AuthConfig{
			APIKeys: []gateway.APIKey{
				{Name: "my-app", Key: "dev-key-12345"},
				{Name: "dashboard", Key: "dashboard-key-67890"},
			},
		},
		Provider: gateway.ProviderConfig{
			Kind: "concourse",
			Concourse: &gateway.ConcourseConfig{
				URL:      "http://localhost:9001",
				Team:     "main",
				Username: "admin",
				Password: "admin",
				// Optionally use a pre-configured bearer token
				// BearerToken: "your-token-here",
				TokenRefreshMargin: 5 * time.Minute,
			},
		},
		Jobs: jobs,
		Logging: gateway.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Create gateway instance
	gw, err := gateway.New(cfg)
	if err != nil {
		log.Fatalf("failed to create gateway: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start the gateway (blocking)
	log.Println("starting gateway on :8080")
	if err := gw.Start(ctx); err != nil {
		log.Fatalf("gateway error: %v", err)
	}

	log.Println("gateway shutdown complete")
}
