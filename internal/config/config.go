package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the gateway configuration
type Config struct {
	Server    ServerConfig
	Auth      AuthConfig
	Concourse ConcourseConfig
	Logging   LoggingConfig
	JobsFile  string
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	APIKeys []APIKey
}

// APIKey represents an API key for authentication
type APIKey struct {
	Name string
	Key  string
}

// ConcourseConfig contains Concourse connection settings
type ConcourseConfig struct {
	URL                string
	Team               string
	Username           string
	Password           string
	BearerToken        string        // Optional: Use pre-configured token
	TokenRefreshMargin time.Duration
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json or text
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Server configuration
	port, err := getEnvInt("SERVER_PORT", 8081)
	if err != nil {
		return nil, fmt.Errorf("parse SERVER_PORT: %w", err)
	}
	cfg.Server.Port = port

	readTimeout, err := getEnvDuration("SERVER_READ_TIMEOUT", "30s")
	if err != nil {
		return nil, fmt.Errorf("parse SERVER_READ_TIMEOUT: %w", err)
	}
	cfg.Server.ReadTimeout = readTimeout

	writeTimeout, err := getEnvDuration("SERVER_WRITE_TIMEOUT", "30s")
	if err != nil {
		return nil, fmt.Errorf("parse SERVER_WRITE_TIMEOUT: %w", err)
	}
	cfg.Server.WriteTimeout = writeTimeout

	// Authentication configuration
	apiKeys, err := parseAPIKeys(os.Getenv("API_KEYS"))
	if err != nil {
		return nil, fmt.Errorf("parse API_KEYS: %w", err)
	}
	cfg.Auth.APIKeys = apiKeys

	// Concourse configuration
	cfg.Concourse.URL = getEnv("CONCOURSE_URL", "")
	if cfg.Concourse.URL == "" {
		return nil, fmt.Errorf("CONCOURSE_URL is required")
	}

	cfg.Concourse.Team = getEnv("CONCOURSE_TEAM", "main")
	cfg.Concourse.Username = getEnv("CONCOURSE_USERNAME", "")
	cfg.Concourse.Password = getEnv("CONCOURSE_PASSWORD", "")
	cfg.Concourse.BearerToken = getEnv("CONCOURSE_BEARER_TOKEN", "")

	refreshMargin, err := getEnvDuration("CONCOURSE_TOKEN_REFRESH_MARGIN", "5m")
	if err != nil {
		return nil, fmt.Errorf("parse CONCOURSE_TOKEN_REFRESH_MARGIN: %w", err)
	}
	cfg.Concourse.TokenRefreshMargin = refreshMargin

	// Logging configuration
	cfg.Logging.Level = getEnv("LOG_LEVEL", "info")
	cfg.Logging.Format = getEnv("LOG_FORMAT", "json")

	// Jobs file
	cfg.JobsFile = getEnv("JOBS_FILE", "configs/jobs.yaml")

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

// getEnvDuration gets a duration environment variable with a default value
func getEnvDuration(key, defaultValue string) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return time.ParseDuration(value)
}

// parseAPIKeys parses comma-separated API keys in format "name:key,name:key"
func parseAPIKeys(value string) ([]APIKey, error) {
	if value == "" {
		return nil, fmt.Errorf("API_KEYS is required")
	}

	var keys []APIKey
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid API key format: %s (expected name:key)", pair)
		}
		keys = append(keys, APIKey{
			Name: strings.TrimSpace(parts[0]),
			Key:  strings.TrimSpace(parts[1]),
		})
	}

	return keys, nil
}
