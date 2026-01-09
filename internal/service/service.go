package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/internal/provider"
	"github.com/lei/simple-ci/internal/provider/concourse"
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
}

// NewService creates a new service instance
func NewService(jobs []*models.Job, prov provider.Provider) *Service {
	jobMap := make(map[string]*models.Job)
	for _, j := range jobs {
		jobMap[j.JobID] = j
	}

	return &Service{
		jobs:     jobMap,
		provider: prov,
	}
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
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, ErrJobNotFound
	}

	// Convert job to provider-specific JobRef
	jobRef, err := s.buildJobRef(job)
	if err != nil {
		return nil, fmt.Errorf("build job ref: %w", err)
	}

	// Trigger via provider
	runRef, err := s.provider.Trigger(ctx, jobRef, provider.TriggerParams{
		Parameters:     params,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("trigger run: %w", err)
	}

	// Get initial status
	providerRun, err := s.provider.GetRun(ctx, runRef)
	if err != nil {
		return nil, fmt.Errorf("get run status: %w", err)
	}

	// Add job_id to the run
	providerRun.JobID = jobID

	return providerRun, nil
}

// GetRun retrieves the status of a run
func (s *Service) GetRun(ctx context.Context, runID string) (*models.Run, error) {
	// Parse run_id to provider-specific RunRef
	runRef, err := s.parseRunRef(runID)
	if err != nil {
		return nil, ErrRunNotFound
	}

	providerRun, err := s.provider.GetRun(ctx, runRef)
	if err != nil {
		if errors.Is(err, provider.ErrRunNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	return providerRun, nil
}

// StreamRunEvents streams events for a run
func (s *Service) StreamRunEvents(ctx context.Context, runID string, writer io.Writer) error {
	runRef, err := s.parseRunRef(runID)
	if err != nil {
		return ErrRunNotFound
	}

	return s.provider.StreamEvents(ctx, runRef, writer)
}

// CancelRun cancels a running build
func (s *Service) CancelRun(ctx context.Context, runID string) error {
	runRef, err := s.parseRunRef(runID)
	if err != nil {
		return ErrRunNotFound
	}

	return s.provider.Cancel(ctx, runRef)
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
