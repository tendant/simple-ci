package models

import "time"

// Job represents a runnable CI job
type Job struct {
	JobID       string            `json:"job_id"`
	Project     string            `json:"project"`
	DisplayName string            `json:"display_name"`
	Environment string            `json:"environment"`
	Provider    JobProviderConfig `json:"provider"`
}

// JobProviderConfig contains provider-specific configuration
type JobProviderConfig struct {
	Kind string                 `json:"kind"` // "concourse", "github", etc.
	Ref  map[string]interface{} `json:"ref"`  // Provider-specific configuration
}

// Run represents a single execution of a job
type Run struct {
	RunID      string     `json:"run_id"`
	JobID      string     `json:"job_id,omitempty"`
	Status     RunStatus  `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// RunStatus represents the state of a run
type RunStatus string

const (
	StatusQueued    RunStatus = "queued"
	StatusRunning   RunStatus = "running"
	StatusSucceeded RunStatus = "succeeded"
	StatusFailed    RunStatus = "failed"
	StatusCanceled  RunStatus = "canceled"
	StatusErrored   RunStatus = "errored"
	StatusUnknown   RunStatus = "unknown"
)

// Event represents a streaming event from a run
type Event struct {
	Type      EventType              `json:"-"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventType represents the type of streaming event
type EventType string

const (
	EventTypeStatus EventType = "status"
	EventTypeLog    EventType = "log"
	EventTypeError  EventType = "error"
)
