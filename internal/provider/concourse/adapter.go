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
)

// Adapter implements the Provider interface for Concourse
type Adapter struct {
	client *Client
	config *Config
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
func NewAdapter(cfg *Config) (*Adapter, error) {
	tokenManager := NewTokenManager(
		cfg.URL,
		cfg.Team,
		cfg.Username,
		cfg.Password,
		cfg.BearerToken,
		cfg.TokenRefreshMargin,
	)
	client := NewClient(cfg.URL, tokenManager)

	return &Adapter{
		client: client,
		config: cfg,
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

// Trigger implements Provider.Trigger
func (a *Adapter) Trigger(ctx context.Context, jobRef provider.JobRef, params provider.TriggerParams) (provider.RunRef, error) {
	ref, ok := jobRef.(*ConcourseJobRef)
	if !ok {
		return nil, fmt.Errorf("invalid job ref type: expected ConcourseJobRef")
	}

	// Trigger build via Concourse API
	build, err := a.client.CreateBuild(ctx, ref.Team, ref.Pipeline, ref.Job, params.Parameters)
	if err != nil {
		return nil, fmt.Errorf("create build: %w", err)
	}

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
	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		return nil, fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	build, err := a.client.GetBuild(ctx, ref.BuildID)
	if err != nil {
		return nil, err
	}

	return mapBuildToRun(build, ref), nil
}

// StreamEvents implements Provider.StreamEvents
func (a *Adapter) StreamEvents(ctx context.Context, runRef provider.RunRef, writer io.Writer) error {
	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		return fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	return a.client.StreamBuildEvents(ctx, ref.BuildID, writer)
}

// Cancel implements Provider.Cancel
func (a *Adapter) Cancel(ctx context.Context, runRef provider.RunRef) error {
	ref, ok := runRef.(*ConcourseRunRef)
	if !ok {
		return fmt.Errorf("invalid run ref type: expected ConcourseRunRef")
	}

	return a.client.AbortBuild(ctx, ref.BuildID)
}
