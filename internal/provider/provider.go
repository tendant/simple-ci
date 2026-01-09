package provider

import (
	"context"
	"io"

	"github.com/lei/simple-ci/internal/models"
)

// Provider abstracts CI backend operations
type Provider interface {
	// Trigger starts a new run for the given job
	// Returns the provider's run reference
	Trigger(ctx context.Context, jobRef JobRef, params TriggerParams) (RunRef, error)

	// GetRun retrieves current status of a run
	GetRun(ctx context.Context, runRef RunRef) (*models.Run, error)

	// StreamEvents streams run events (logs, status changes) as SSE
	// Writes directly to the provided writer
	StreamEvents(ctx context.Context, runRef RunRef, writer io.Writer) error

	// Cancel aborts a running build
	Cancel(ctx context.Context, runRef RunRef) error
}

// JobRef is a provider-specific job reference
type JobRef interface {
	Kind() string // "concourse", "github", etc.
}

// RunRef is a provider-specific run identifier
type RunRef interface {
	Kind() string
	ID() string // The actual run_id exposed to API clients
}

// TriggerParams contains parameters for triggering a run
type TriggerParams struct {
	Parameters     map[string]interface{} // User-provided params
	IdempotencyKey string                 // Optional
}
