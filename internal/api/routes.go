package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter creates and configures the HTTP router
func NewRouter(handlers *Handlers, authMiddleware *AuthMiddleware, loggingMiddleware *LoggingMiddleware) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware - ORDER MATTERS!
	r.Use(middleware.RequestID)      // Generate request ID first
	r.Use(middleware.RealIP)         // Extract real IP
	r.Use(loggingMiddleware.Handler) // Add logger to context with request ID
	r.Use(middleware.Recoverer)      // Panic recovery
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"}, // Expose request ID
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check endpoint (no auth required)
	r.Get("/health", handlers.Health)

	// API v1 routes (with authentication)
	r.Route("/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)

		// Jobs
		r.Get("/jobs", handlers.ListJobs)
		r.Post("/jobs/{job_id}/runs", handlers.TriggerRun)

		// Runs
		r.Get("/runs/{run_id}", handlers.GetRun)
		r.Get("/runs/{run_id}/events", handlers.StreamEvents)
		r.Post("/runs/{run_id}/cancel", handlers.CancelRun)

		// Builds - detailed build information
		r.Get("/builds/{build_id}", handlers.GetBuildDetails)

		// Discovery - list pipelines and jobs from provider
		r.Get("/discovery/pipelines", handlers.ListPipelines)
		r.Get("/discovery/pipelines/{pipeline}/jobs", handlers.ListPipelineJobs)
		r.Get("/discovery/pipelines/{pipeline}/jobs/{job}/builds", handlers.ListJobBuilds)
	})

	return r
}
