// Package gateway provides a reusable CI Gateway library that can be embedded
// into other Go applications.
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lei/simple-ci/internal/api"
	"github.com/lei/simple-ci/internal/config"
	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/internal/provider"
	"github.com/lei/simple-ci/internal/provider/concourse"
	"github.com/lei/simple-ci/internal/service"
	"github.com/lei/simple-ci/pkg/logger"
)

// Gateway represents a Simple CI Gateway instance that can be embedded in applications
type Gateway struct {
	config  *Config
	service *service.Service
	router  http.Handler
	server  *http.Server
	logger  *logger.Logger
}

// Config holds the configuration for the Gateway
type Config struct {
	// Server configuration
	Server ServerConfig

	// Authentication configuration
	Auth AuthConfig

	// Provider configuration (currently supports Concourse)
	Provider ProviderConfig

	// Jobs configuration
	Jobs []*models.Job

	// Logger configuration
	Logging LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// APIKeys is a list of API keys for authentication
	APIKeys []APIKey
}

// APIKey represents an API key for authentication
type APIKey struct {
	Name string
	Key  string
}

// ProviderConfig holds CI provider configuration
type ProviderConfig struct {
	Kind string // Currently only "concourse" is supported

	// Concourse-specific configuration
	Concourse *ConcourseConfig
}

// ConcourseConfig holds Concourse CI specific configuration
type ConcourseConfig struct {
	URL                string
	Team               string
	Username           string
	Password           string
	BearerToken        string
	TokenRefreshMargin time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json or text
}

// New creates a new Gateway instance with the provided configuration
func New(cfg *Config) (*Gateway, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Initialize logger
	appLogger := logger.New(cfg.Logging.Level, cfg.Logging.Format)

	// Initialize provider
	var prov provider.Provider
	var err error

	switch cfg.Provider.Kind {
	case "concourse":
		if cfg.Provider.Concourse == nil {
			return nil, fmt.Errorf("concourse configuration required when provider kind is 'concourse'")
		}
		providerCfg := &concourse.Config{
			URL:                cfg.Provider.Concourse.URL,
			Team:               cfg.Provider.Concourse.Team,
			Username:           cfg.Provider.Concourse.Username,
			Password:           cfg.Provider.Concourse.Password,
			BearerToken:        cfg.Provider.Concourse.BearerToken,
			TokenRefreshMargin: cfg.Provider.Concourse.TokenRefreshMargin,
		}
		prov, err = concourse.NewAdapter(providerCfg, appLogger)
		if err != nil {
			return nil, fmt.Errorf("initialize concourse provider: %w", err)
		}
		appLogger.Info("initialized concourse provider", "url", cfg.Provider.Concourse.URL, "team", cfg.Provider.Concourse.Team)

	default:
		return nil, fmt.Errorf("unsupported provider kind: %s", cfg.Provider.Kind)
	}

	// Initialize service layer
	svc := service.NewService(cfg.Jobs, prov, appLogger)

	// Initialize API layer
	handlers := api.NewHandlers(svc)

	// Convert APIKeys to internal config format
	configAPIKeys := make([]config.APIKey, len(cfg.Auth.APIKeys))
	for i, key := range cfg.Auth.APIKeys {
		configAPIKeys[i] = config.APIKey{
			Name: key.Name,
			Key:  key.Key,
		}
	}
	authMiddleware := api.NewAuthMiddleware(configAPIKeys)
	loggingMiddleware := api.NewLoggingMiddleware(appLogger)
	router := api.NewRouter(handlers, authMiddleware, loggingMiddleware)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &Gateway{
		config:  cfg,
		service: svc,
		router:  router,
		server:  srv,
		logger:  appLogger,
	}, nil
}

// Start starts the HTTP server
// This is a blocking call that will run until the context is canceled or an error occurs
func (g *Gateway) Start(ctx context.Context) error {
	serverErrors := make(chan error, 1)

	// Start server in goroutine
	go func() {
		g.logger.Info("starting http server", "port", g.config.Server.Port)
		serverErrors <- g.server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil

	case <-ctx.Done():
		g.logger.Info("shutdown signal received")

		// Graceful shutdown with 30s timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := g.server.Shutdown(shutdownCtx); err != nil {
			g.server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		g.logger.Info("server stopped gracefully")
		return nil
	}
}

// Handler returns the http.Handler for the gateway
// Use this if you want to integrate the gateway into an existing HTTP server
func (g *Gateway) Handler() http.Handler {
	return g.router
}

// Service returns the underlying service layer
// Use this for direct programmatic access to gateway functionality
func (g *Gateway) Service() *service.Service {
	return g.service
}

// NewFromEnv creates a Gateway instance from environment variables and config files
// This is a convenience function that mirrors the behavior of the standalone gateway
func NewFromEnv(jobsFile string) (*Gateway, error) {
	// Load gateway configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Load job definitions
	jobs, err := config.LoadJobs(jobsFile)
	if err != nil {
		return nil, fmt.Errorf("load jobs: %w", err)
	}

	// Convert to Gateway config
	// Convert APIKeys from internal config format
	gwAPIKeys := make([]APIKey, len(cfg.Auth.APIKeys))
	for i, key := range cfg.Auth.APIKeys {
		gwAPIKeys[i] = APIKey{
			Name: key.Name,
			Key:  key.Key,
		}
	}

	gwConfig := &Config{
		Server: ServerConfig{
			Port:         cfg.Server.Port,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
		Auth: AuthConfig{
			APIKeys: gwAPIKeys,
		},
		Provider: ProviderConfig{
			Kind: "concourse",
			Concourse: &ConcourseConfig{
				URL:                cfg.Concourse.URL,
				Team:               cfg.Concourse.Team,
				Username:           cfg.Concourse.Username,
				Password:           cfg.Concourse.Password,
				BearerToken:        cfg.Concourse.BearerToken,
				TokenRefreshMargin: cfg.Concourse.TokenRefreshMargin,
			},
		},
		Jobs: jobs,
		Logging: LoggingConfig{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
		},
	}

	return New(gwConfig)
}
