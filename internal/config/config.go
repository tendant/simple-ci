package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the gateway configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Auth      AuthConfig      `yaml:"auth"`
	Concourse ConcourseConfig `yaml:"concourse"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	APIKeys []APIKey `yaml:"api_keys"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// ConcourseConfig contains Concourse connection settings
type ConcourseConfig struct {
	URL                string        `yaml:"url"`
	Team               string        `yaml:"team"`
	Username           string        `yaml:"username"`
	Password           string        `yaml:"password"`
	BearerToken        string        `yaml:"bearer_token"`        // Optional: Use pre-configured token
	TokenRefreshMargin time.Duration `yaml:"token_refresh_margin"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json or text
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Concourse.TokenRefreshMargin == 0 {
		cfg.Concourse.TokenRefreshMargin = 5 * time.Minute
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}

	return &cfg, nil
}
