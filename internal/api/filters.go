package api

import (
	"strings"

	"github.com/lei/simple-ci/internal/provider/concourse"
)

// FilterPipelines filters pipelines based on query parameters
func FilterPipelines(pipelines []concourse.Pipeline, search string, paused, archived *bool) []concourse.Pipeline {
	if search == "" && paused == nil && archived == nil {
		return pipelines
	}

	filtered := make([]concourse.Pipeline, 0, len(pipelines))
	searchLower := strings.ToLower(search)

	for _, p := range pipelines {
		// Search filter
		if search != "" && !strings.Contains(strings.ToLower(p.Name), searchLower) {
			continue
		}

		// Paused filter
		if paused != nil && p.Paused != *paused {
			continue
		}

		// Archived filter
		if archived != nil && p.Archived != *archived {
			continue
		}

		filtered = append(filtered, p)
	}

	return filtered
}

// FilterJobs filters jobs based on query parameters
func FilterJobs(jobs []concourse.Job, search string, paused *bool) []concourse.Job {
	if search == "" && paused == nil {
		return jobs
	}

	filtered := make([]concourse.Job, 0, len(jobs))
	searchLower := strings.ToLower(search)

	for _, j := range jobs {
		// Search filter
		if search != "" && !strings.Contains(strings.ToLower(j.Name), searchLower) {
			continue
		}

		// Paused filter
		if paused != nil && j.Paused != *paused {
			continue
		}

		filtered = append(filtered, j)
	}

	return filtered
}

// parseBoolParam parses boolean query parameters
func parseBoolParam(value string) *bool {
	if value == "" {
		return nil
	}

	if value == "true" || value == "1" {
		result := true
		return &result
	}

	if value == "false" || value == "0" {
		result := false
		return &result
	}

	return nil
}
