package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/pkg/gateway"
)

func main() {
	// Define jobs
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
			Format: "text",
		},
	}

	// Create gateway instance
	gw, err := gateway.New(cfg)
	if err != nil {
		log.Fatalf("failed to create gateway: %v", err)
	}

	// Get direct access to the service layer
	svc := gw.Service()

	ctx := context.Background()

	// List all jobs
	allJobs := svc.ListJobs(ctx)
	fmt.Printf("Available jobs: %d\n", len(allJobs))
	for _, job := range allJobs {
		fmt.Printf("  - %s: %s (%s)\n", job.JobID, job.DisplayName, job.Environment)
	}

	// Trigger a run programmatically
	fmt.Println("\nTriggering job...")
	run, err := svc.TriggerRun(ctx, "job_example_hello", map[string]interface{}{
		"git_sha":     "abc123",
		"environment": "dev",
	}, "")
	if err != nil {
		log.Fatalf("failed to trigger run: %v", err)
	}

	fmt.Printf("Run triggered: %s (status: %s)\n", run.RunID, run.Status)

	// Poll run status
	fmt.Println("\nPolling run status...")
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)

		runStatus, err := svc.GetRun(ctx, run.RunID)
		if err != nil {
			log.Fatalf("failed to get run status: %v", err)
		}

		fmt.Printf("  Status: %s\n", runStatus.Status)

		if runStatus.Status == "succeeded" || runStatus.Status == "failed" {
			fmt.Printf("Run completed with status: %s\n", runStatus.Status)
			break
		}
	}
}
