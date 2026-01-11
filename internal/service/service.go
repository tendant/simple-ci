package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/internal/provider"
	"github.com/lei/simple-ci/internal/provider/concourse"
	"github.com/lei/simple-ci/pkg/logger"
)

var (
	// ErrJobNotFound indicates the requested job doesn't exist
	ErrJobNotFound = errors.New("job not found")
	// ErrRunNotFound indicates the requested run doesn't exist
	ErrRunNotFound = errors.New("run not found")
)

// Service coordinates business logic between API and provider layers
type Service struct {
	jobs     map[string]*models.Job
	provider provider.Provider
	logger   *logger.Logger
}

// NewService creates a new service instance
func NewService(jobs []*models.Job, prov provider.Provider, log *logger.Logger) *Service {
	jobMap := make(map[string]*models.Job)
	for _, j := range jobs {
		jobMap[j.JobID] = j
	}

	return &Service{
		jobs:     jobMap,
		provider: prov,
		logger:   log,
	}
}

// getLogger retrieves logger from context or falls back to service logger
func (s *Service) getLogger(ctx context.Context) *logger.Logger {
	// Try to get request-scoped logger from context
	// Using plain string key for cross-package compatibility
	if ctxLogger, ok := ctx.Value("logger").(*logger.Logger); ok {
		return ctxLogger
	}
	// Fallback to service logger
	return s.logger
}

// ListJobs returns all configured jobs
func (s *Service) ListJobs(ctx context.Context) []*models.Job {
	jobs := make([]*models.Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}

// TriggerRun triggers a new run for the specified job
func (s *Service) TriggerRun(ctx context.Context, jobID string, params map[string]interface{}, idempotencyKey string) (*models.Run, error) {
	logger := s.getLogger(ctx)

	logger.Debug("service: triggering run",
		"job_id", jobID,
		"param_count", len(params),
		"has_idempotency_key", idempotencyKey != "")

	job, exists := s.jobs[jobID]
	if !exists {
		logger.Debug("service: job not found", "job_id", jobID)
		return nil, ErrJobNotFound
	}

	// Convert job to provider-specific JobRef
	logger.Debug("service: building job ref",
		"job_id", jobID,
		"provider_kind", job.Provider.Kind)
	jobRef, err := s.buildJobRef(job)
	if err != nil {
		logger.Error("service: failed to build job ref",
			"job_id", jobID,
			"error", err)
		return nil, fmt.Errorf("build job ref: %w", err)
	}

	// Trigger via provider
	logger.Debug("service: calling provider trigger", "job_id", jobID)
	runRef, err := s.provider.Trigger(ctx, jobRef, provider.TriggerParams{
		Parameters:     params,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		logger.Error("service: provider trigger failed",
			"job_id", jobID,
			"error", err)
		return nil, fmt.Errorf("trigger run: %w", err)
	}

	// Get initial status
	logger.Debug("service: fetching initial run status", "job_id", jobID)
	providerRun, err := s.provider.GetRun(ctx, runRef)
	if err != nil {
		logger.Error("service: failed to get run status",
			"job_id", jobID,
			"error", err)
		return nil, fmt.Errorf("get run status: %w", err)
	}

	// Add job_id to the run
	providerRun.JobID = jobID

	logger.Info("service: run triggered successfully",
		"job_id", jobID,
		"run_id", providerRun.RunID,
		"status", providerRun.Status)

	return providerRun, nil
}

// GetRun retrieves the status of a run
func (s *Service) GetRun(ctx context.Context, runID string) (*models.Run, error) {
	logger := s.getLogger(ctx)

	logger.Debug("service: getting run status", "run_id", runID)

	// Parse run_id to provider-specific RunRef
	runRef, err := s.parseRunRef(runID)
	if err != nil {
		logger.Debug("service: failed to parse run_id", "run_id", runID, "error", err)
		return nil, ErrRunNotFound
	}

	providerRun, err := s.provider.GetRun(ctx, runRef)
	if err != nil {
		if errors.Is(err, provider.ErrRunNotFound) {
			logger.Debug("service: run not found in provider", "run_id", runID)
			return nil, ErrRunNotFound
		}
		logger.Error("service: provider get run failed", "run_id", runID, "error", err)
		return nil, err
	}

	logger.Debug("service: run status retrieved",
		"run_id", runID,
		"status", providerRun.Status)

	return providerRun, nil
}

// StreamRunEvents streams events for a run
func (s *Service) StreamRunEvents(ctx context.Context, runID string, writer io.Writer) error {
	logger := s.getLogger(ctx)

	logger.Info("service: starting event stream", "run_id", runID)

	runRef, err := s.parseRunRef(runID)
	if err != nil {
		logger.Debug("service: failed to parse run_id for streaming", "run_id", runID, "error", err)
		return ErrRunNotFound
	}

	err = s.provider.StreamEvents(ctx, runRef, writer)
	if err != nil {
		logger.Error("service: event stream failed", "run_id", runID, "error", err)
		return err
	}

	logger.Info("service: event stream completed", "run_id", runID)
	return nil
}

// CancelRun cancels a running build
func (s *Service) CancelRun(ctx context.Context, runID string) error {
	logger := s.getLogger(ctx)

	logger.Info("service: canceling run", "run_id", runID)

	runRef, err := s.parseRunRef(runID)
	if err != nil {
		logger.Debug("service: failed to parse run_id for cancel", "run_id", runID, "error", err)
		return ErrRunNotFound
	}

	err = s.provider.Cancel(ctx, runRef)
	if err != nil {
		logger.Error("service: cancel run failed", "run_id", runID, "error", err)
		return err
	}

	logger.Info("service: run canceled successfully", "run_id", runID)
	return nil
}

// buildJobRef converts a Job to a provider-specific JobRef
func (s *Service) buildJobRef(job *models.Job) (provider.JobRef, error) {
	switch job.Provider.Kind {
	case "concourse":
		// Extract Concourse-specific fields from provider ref
		team, ok := job.Provider.Ref["team"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'team' in concourse job ref")
		}
		pipeline, ok := job.Provider.Ref["pipeline"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'pipeline' in concourse job ref")
		}
		jobName, ok := job.Provider.Ref["job"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'job' in concourse job ref")
		}

		return &concourse.ConcourseJobRef{
			Team:     team,
			Pipeline: pipeline,
			Job:      jobName,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider kind: %s", job.Provider.Kind)
	}
}

// parseRunRef parses a run_id string to a provider-specific RunRef
func (s *Service) parseRunRef(runID string) (provider.RunRef, error) {
	// In v1, assume all runs are Concourse
	// Format: team/pipeline/job/build_id
	return concourse.ParseRunRef(runID)
}
