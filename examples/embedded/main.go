package main

import (
	"context"
	"log"
	"net/http"
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

	// Create gateway configuration (without server config - we'll use our own)
	cfg := &gateway.Config{
		Server: gateway.ServerConfig{
			Port:         8080, // Not used when embedding
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
			Format: "json",
		},
	}

	// Create gateway instance
	gw, err := gateway.New(cfg)
	if err != nil {
		log.Fatalf("failed to create gateway: %v", err)
	}

	// Create custom HTTP router
	mux := http.NewServeMux()

	// Mount CI Gateway under /ci path
	mux.Handle("/ci/", http.StripPrefix("/ci", gw.Handler()))

	// Add custom application endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to my application!\nCI Gateway is available at /ci/"))
	})

	mux.HandleFunc("/custom", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Custom endpoint"))
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start server in goroutine
	go func() {
		log.Println("starting server on :8080")
		log.Println("  - Custom endpoints: http://localhost:8080/")
		log.Println("  - CI Gateway:       http://localhost:8080/ci/")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	log.Println("server shutdown complete")
}
