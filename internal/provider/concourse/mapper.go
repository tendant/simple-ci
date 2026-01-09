package concourse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lei/simple-ci/internal/models"
	"github.com/lei/simple-ci/internal/provider"
)

// mapBuildToRun converts a Concourse build to a generic Run
func mapBuildToRun(build *Build, runRef *ConcourseRunRef) *models.Run {
	run := &models.Run{
		RunID:     runRef.ID(),
		Status:    mapStatus(build.Status),
		CreatedAt: time.Unix(build.CreateTime, 0),
	}

	if build.StartTime > 0 {
		startedAt := time.Unix(build.StartTime, 0)
		run.StartedAt = &startedAt
	}

	if build.EndTime > 0 {
		finishedAt := time.Unix(build.EndTime, 0)
		run.FinishedAt = &finishedAt
	}

	return run
}

// mapStatus converts Concourse build status to generic RunStatus
func mapStatus(concourseStatus string) models.RunStatus {
	switch concourseStatus {
	case "pending":
		return models.StatusQueued
	case "started":
		return models.StatusRunning
	case "succeeded":
		return models.StatusSucceeded
	case "failed":
		return models.StatusFailed
	case "aborted":
		return models.StatusCanceled
	case "errored":
		return models.StatusErrored
	default:
		return models.StatusUnknown
	}
}

// parseError converts HTTP error responses to provider errors
func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return provider.ErrRunNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return provider.ErrUnauthorized
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return provider.ErrProviderUnavailable
	default:
		var errResp struct {
			Error string `json:"error"`
		}

		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return &provider.ProviderError{
				Code:    resp.StatusCode,
				Message: errResp.Error,
			}
		}

		return &provider.ProviderError{
			Code:    resp.StatusCode,
			Message: string(body),
		}
	}
}

// parseConcourseEvent transforms Concourse SSE events to generic events
func parseConcourseEvent(line string) (string, error) {
	// Concourse sends events as newline-delimited JSON
	// For now, we'll pass them through as-is
	// In a more complete implementation, we'd parse and transform specific event types

	// Simple passthrough for MVP
	if line == "" {
		return "", nil
	}

	return fmt.Sprintf("data: %s\n\n", line), nil
}
