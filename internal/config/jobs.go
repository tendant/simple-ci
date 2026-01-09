package config

import (
	"fmt"
	"os"

	"github.com/lei/simple-ci/internal/models"
	"gopkg.in/yaml.v3"
)

// JobsConfig represents the jobs configuration file structure
type JobsConfig struct {
	Jobs []JobDefinition `yaml:"jobs"`
}

// JobDefinition represents a job definition in the config file
type JobDefinition struct {
	JobID       string                 `yaml:"job_id"`
	Project     string                 `yaml:"project"`
	DisplayName string                 `yaml:"display_name"`
	Environment string                 `yaml:"environment"`
	Provider    ProviderConfig         `yaml:"provider"`
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	Kind string                 `yaml:"kind"`
	Ref  map[string]interface{} `yaml:"ref"`
}

// LoadJobs reads and parses the jobs configuration file
func LoadJobs(path string) ([]*models.Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read jobs config file: %w", err)
	}

	var cfg JobsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse jobs config: %w", err)
	}

	// Validate and convert to models
	jobs := make([]*models.Job, 0, len(cfg.Jobs))
	for i, jd := range cfg.Jobs {
		if jd.JobID == "" {
			return nil, fmt.Errorf("job at index %d missing job_id", i)
		}
		if jd.Provider.Kind == "" {
			return nil, fmt.Errorf("job %s missing provider kind", jd.JobID)
		}

		jobs = append(jobs, &models.Job{
			JobID:       jd.JobID,
			Project:     jd.Project,
			DisplayName: jd.DisplayName,
			Environment: jd.Environment,
			Provider: models.JobProviderConfig{
				Kind: jd.Provider.Kind,
				Ref:  jd.Provider.Ref,
			},
		})
	}

	return jobs, nil
}
