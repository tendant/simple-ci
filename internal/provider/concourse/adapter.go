package concourse

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/internal/provider"
	"github.com/lei/simple-ci/pkg/logger"
)

// Adapter implements the Provider interface for Concourse
type Adapter struct {
	client *Client
	config *Config
	logger *logger.Logger
}

// Config contains Concourse connection settings
type Config struct {
	URL                string
	Team               string
	Username           string
	Password           string
	BearerToken        string
	TokenRefreshMargin time.Duration
}

// NewAdapter creates a new Concourse adapter
func NewAdapter(cfg *Config, log *logger.Logger) (*Adapter, error) {
	tokenManager := NewTokenManager(
		cfg.URL,
		cfg.Team,
		cfg.Username,
		cfg.Password,
		cfg.BearerToken,
		cfg.TokenRefreshMargin,
		log,
	)
	client := NewClient(cfg.URL, tokenManager, log)

	return &Adapter{
		client: client,
		config: cfg,
		logger: log,
	}, nil
}

// ConcourseJobRef represents a Concourse job reference
type ConcourseJobRef struct {
	Team     string
	Pipeline string
	Job      string
}

func (c *ConcourseJobRef) Kind() string {
	return "concourse"
}

// ConcourseRunRef represents a Concourse run reference
type ConcourseRunRef struct {
	Team      string
	Pipeline  string
	Job       string
	BuildID   int
	BuildName string
}

func (c *ConcourseRunRef) Kind() string {
	return "concourse"
}

func (c *ConcourseRunRef) ID() string {
	// Format: team:pipeline:job:build_id (URL-safe)
	return fmt.Sprintf("%s:%s:%s:%d", c.Team, c.Pipeline, c.Job, c.BuildID)
}

// ParseRunRef parses a run_id string back to ConcourseRunRef
func ParseRunRef(runID string) (*ConcourseRunRef, error) {
	parts := strings.Split(runID, ":")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid run_id format, expected team:pipeline:job:build_id")
	}

	buildID, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid build_id in run_id: %w", err)
	}

	return &ConcourseRunRef{
		Team:     parts[0],
		Pipeline: parts[1],
		Job:      parts[2],
		BuildID:  buildID,
	}, nil
}

// getLogger retrieves logger from context or falls back to adapter logger
func (a *Adapter) getLogger(ctx context.Context) *logger.Logger {
	// Try to get request-scoped logger from context
	if ctxLogger, ok := ctx.Value("logger").(*logger.Logger); ok {
		return ctxLogger
	}
	// Fallback to adapter logger
	return a.logger
}

// Trigger implements Provider.Trigger
func (a *Adapter) Trigger(ctx context.Context, jobRef provider.JobRef, params provider.TriggerParams) (provider.RunRef, error) {
	logger := a.getLogger(ctx)

	ref, ok := jobRef.(*ConcourseJobRef)
	if !ok {
		logger.Error("provider: invalid job ref type", "expected", "ConcourseJobRef")
		return nil, fmt.Errorf("invalid job ref type: expected ConcourseJobRef")
	}

	logger.Debug("provider: triggering concourse build",
		"team", ref.Team,
		"pipeline", ref.Pipeline,
		"job", ref.Job,
		"param_count", len(params.Parameters))

	// Trigger build via Concourse API
	build, err := a.client.CreateBuild(ctx, ref.Team, ref.Pipeline, ref.Job, params.Parameters)
	if err != nil {
		logger.Error("provider: failed to create build",
			"team", ref.Team,
			"pipeline", ref.Pipeline,
			"job", ref.Job,
			"error", err)
		return nil, fmt.Errorf("create build: %w", err)
	}

	logger.Info("provider: build triggered",
		"team", ref.Team,
		"pipeline", ref.Pipeline,
		"job", ref.Job,
		"build_id", build.ID,
		"build_name", build.Name)

	return &ConcourseRunRef{
		Team:      ref.Team,
		Pipeline:  ref.Pipeline,
		Job:       ref.Job,
		BuildID:   build.ID,
		BuildName: build.Name,
	}, nil
}

// GetRun implements Provider.GetRun
func (a *Adapter) GetRun(ctx context.Context, runRef provider.RunRef) (*models.Run, error) {
	logger := a.getLogger(ctx)

	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		logger.Error("provider: invalid run ref type", "expected", "ConcourseRunRef")
		return nil, fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	logger.Debug("provider: getting build status",
		"team", ref.Team,
		"pipeline", ref.Pipeline,
		"job", ref.Job,
		"build_id", ref.BuildID)

	build, err := a.client.GetBuild(ctx, ref.BuildID)
	if err != nil {
		logger.Error("provider: failed to get build",
			"build_id", ref.BuildID,
			"error", err)
		return nil, err
	}

	logger.Debug("provider: build status retrieved",
		"build_id", ref.BuildID,
		"status", build.Status)

	return mapBuildToRun(build, ref), nil
}

// StreamEvents implements Provider.StreamEvents
func (a *Adapter) StreamEvents(ctx context.Context, runRef provider.RunRef, writer io.Writer) error {
	logger := a.getLogger(ctx)

	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		logger.Error("provider: invalid run ref type for streaming", "expected", "ConcourseRunRef")
		return fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	logger.Info("provider: starting build event stream",
		"team", ref.Team,
		"pipeline", ref.Pipeline,
		"job", ref.Job,
		"build_id", ref.BuildID)

	err := a.client.StreamBuildEvents(ctx, ref.BuildID, writer)
	if err != nil {
		logger.Error("provider: build event stream failed",
			"build_id", ref.BuildID,
			"error", err)
		return err
	}

	logger.Info("provider: build event stream completed",
		"build_id", ref.BuildID)
	return nil
}

// Cancel implements Provider.Cancel
func (a *Adapter) Cancel(ctx context.Context, runRef provider.RunRef) error {
	logger := a.getLogger(ctx)

	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		logger.Error("provider: invalid run ref type for cancel", "expected", "ConcourseRunRef")
		return fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	logger.Info("provider: aborting build",
		"team", ref.Team,
		"pipeline", ref.Pipeline,
		"job", ref.Job,
		"build_id", ref.BuildID)

	err := a.client.AbortBuild(ctx, ref.BuildID)
	if err != nil {
		logger.Error("provider: failed to abort build",
			"build_id", ref.BuildID,
			"error", err)
		return err
	}

	logger.Info("provider: build aborted successfully",
		"build_id", ref.BuildID)
	return nil
}

// ListPipelines lists all pipelines for the configured team
func (a *Adapter) ListPipelines(ctx context.Context) ([]Pipeline, error) {
	logger := a.getLogger(ctx)

	logger.Debug("provider: listing pipelines",
		"team", a.config.Team)

	pipelines, err := a.client.ListPipelines(ctx, a.config.Team)
	if err != nil {
		logger.Error("provider: failed to list pipelines",
			"team", a.config.Team,
			"error", err)
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	logger.Info("provider: pipelines listed",
		"team", a.config.Team,
		"count", len(pipelines))

	return pipelines, nil
}

// ListJobs lists all jobs in a pipeline
func (a *Adapter) ListJobs(ctx context.Context, pipeline string) ([]Job, error) {
	logger := a.getLogger(ctx)

	logger.Debug("provider: listing jobs",
		"team", a.config.Team,
		"pipeline", pipeline)

	jobs, err := a.client.ListJobs(ctx, a.config.Team, pipeline)
	if err != nil {
		logger.Error("provider: failed to list jobs",
			"team", a.config.Team,
			"pipeline", pipeline,
			"error", err)
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	logger.Info("provider: jobs listed",
		"team", a.config.Team,
		"pipeline", pipeline,
		"count", len(jobs))

	return jobs, nil
}
