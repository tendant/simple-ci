package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lei/simple-ci/internal/api"
	"github.com/lei/simple-ci/internal/config"
	"github.com/lei/simple-ci/internal/provider/concourse"
	"github.com/lei/simple-ci/internal/service"
	"github.com/lei/simple-ci/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

func run() error {
	// Load gateway configuration
	cfg, err := config.Load("configs/gateway.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	appLogger.Info("starting simple-ci gateway")

	// Load job definitions
	jobs, err := config.LoadJobs("configs/jobs.yaml")
	if err != nil {
		return fmt.Errorf("load jobs: %w", err)
	}
	appLogger.Infow("loaded jobs", "count", len(jobs))

	// Initialize Concourse provider
	providerCfg := &concourse.Config{
		URL:                cfg.Concourse.URL,
		Team:               cfg.Concourse.Team,
		Username:           cfg.Concourse.Username,
		Password:           cfg.Concourse.Password,
		BearerToken:        cfg.Concourse.BearerToken,
		TokenRefreshMargin: cfg.Concourse.TokenRefreshMargin,
	}
	provider, err := concourse.NewAdapter(providerCfg)
	if err != nil {
		return fmt.Errorf("initialize provider: %w", err)
	}
	appLogger.Infow("initialized concourse provider", "url", cfg.Concourse.URL, "team", cfg.Concourse.Team)

	// Initialize service layer
	svc := service.NewService(jobs, provider)

	// Initialize API layer
	handlers := api.NewHandlers(svc)
	authMiddleware := api.NewAuthMiddleware(cfg.Auth.APIKeys)
	router := api.NewRouter(handlers, authMiddleware)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Infow("starting http server", "port", cfg.Server.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		appLogger.Infow("shutdown signal received", "signal", sig.String())

		// Graceful shutdown with 30s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		appLogger.Info("server stopped gracefully")
	}

	return nil
}
