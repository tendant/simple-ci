package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/lei/simple-ci/pkg/gateway"
)

func main() {
	// Load .env file (optional - can also use system environment variables)
	_ = godotenv.Load()

	// Create gateway from environment variables
	// This reads configuration from env vars and loads jobs from the specified file
	gw, err := gateway.NewFromEnv("configs/jobs.yaml")
	if err != nil {
		log.Fatalf("failed to create gateway: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start the gateway (blocking)
	log.Println("starting gateway from environment configuration")
	if err := gw.Start(ctx); err != nil {
		log.Fatalf("gateway error: %v", err)
	}

	log.Println("gateway shutdown complete")
}
