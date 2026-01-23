package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/lei/simple-ci/pkg/gateway"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

func run() error {
	// Load .env file (ignore error if file doesn't exist - env vars might be set externally)
	_ = godotenv.Load()

	// Determine jobs file path from environment or use default
	jobsFile := os.Getenv("JOBS_FILE")
	if jobsFile == "" {
		jobsFile = "configs/jobs.yaml"
	}

	// Create gateway from environment configuration
	gw, err := gateway.NewFromEnv(jobsFile)
	if err != nil {
		return err
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start the gateway (blocks until shutdown)
	return gw.Start(ctx)
}
