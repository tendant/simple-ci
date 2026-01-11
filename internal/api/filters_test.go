package api

import (
	"testing"

	"github.com/lei/simple-ci/internal/provider/concourse"
)

func TestFilterPipelines(t *testing.T) {
	pipelines := []concourse.Pipeline{
		{Name: "site-barbiecattleco", TeamName: "main", Paused: false, Archived: false},
		{Name: "site-cottagevillagechildress", TeamName: "main", Paused: false, Archived: false},
		{Name: "old-pipeline", TeamName: "main", Paused: true, Archived: true},
	}

	tests := []struct {
		name     string
		search   string
		paused   *bool
		archived *bool
		want     int
	}{
		{"no filters", "", nil, nil, 3},
		{"search barbie", "barbie", nil, nil, 1},
		{"search cottage", "cottage", nil, nil, 1},
		{"paused false", "", boolPtr(false), nil, 2},
		{"paused true", "", boolPtr(true), nil, 1},
		{"archived false", "", nil, boolPtr(false), 2},
		{"archived true", "", nil, boolPtr(true), 1},
		{"search + archived", "site", nil, boolPtr(false), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPipelines(pipelines, tt.search, tt.paused, tt.archived)
			if len(got) != tt.want {
				t.Errorf("FilterPipelines() = %d pipelines, want %d", len(got), tt.want)
			}
		})
	}
}

func TestFilterJobs(t *testing.T) {
	jobs := []concourse.Job{
		{Name: "deploy-site", PipelineName: "my-pipeline", TeamName: "main", Paused: false},
		{Name: "build-app", PipelineName: "my-pipeline", TeamName: "main", Paused: false},
		{Name: "test-app", PipelineName: "my-pipeline", TeamName: "main", Paused: true},
	}

	tests := []struct {
		name   string
		search string
		paused *bool
		want   int
	}{
		{"no filters", "", nil, 3},
		{"search deploy", "deploy", nil, 1},
		{"search app", "app", nil, 2},
		{"paused false", "", boolPtr(false), 2},
		{"paused true", "", boolPtr(true), 1},
		{"search + paused", "app", boolPtr(false), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterJobs(jobs, tt.search, tt.paused)
			if len(got) != tt.want {
				t.Errorf("FilterJobs() = %d jobs, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseBoolParam(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *bool
	}{
		{"empty", "", nil},
		{"true", "true", boolPtr(true)},
		{"1", "1", boolPtr(true)},
		{"false", "false", boolPtr(false)},
		{"0", "0", boolPtr(false)},
		{"invalid", "invalid", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBoolParam(tt.value)
			if (got == nil) != (tt.want == nil) {
				t.Errorf("parseBoolParam() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("parseBoolParam() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
