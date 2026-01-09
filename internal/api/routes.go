package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter creates and configures the HTTP router
func NewRouter(handlers *Handlers, authMiddleware *AuthMiddleware) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
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
	})

	return r
}
