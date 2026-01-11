package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lei/simple-ci/internal/provider"
	"github.com/lei/simple-ci/internal/service"
)

// Handlers contains HTTP handler functions
type Handlers struct {
	service *service.Service
}

// NewHandlers creates a new handlers instance
func NewHandlers(svc *service.Service) *Handlers {
	return &Handlers{service: svc}
}

// Health handles health check requests
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ListJobs handles GET /v1/jobs
func (h *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.service.ListJobs(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}

// TriggerRun handles POST /v1/jobs/{job_id}/runs
func (h *Handlers) TriggerRun(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	jobID := chi.URLParam(r, "job_id")

	if logger != nil {
		logger.Debug("triggering run", "job_id", jobID)
	}

	var req struct {
		Parameters     map[string]interface{} `json:"parameters"`
		IdempotencyKey string                 `json:"idempotency_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if logger != nil {
			logger.Warn("invalid request body", "error", err)
		}
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if logger != nil {
		logger.Debug("decoded trigger request",
			"job_id", jobID,
			"has_parameters", len(req.Parameters) > 0,
			"has_idempotency_key", req.IdempotencyKey != "")
	}

	run, err := h.service.TriggerRun(r.Context(), jobID, req.Parameters, req.IdempotencyKey)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("run triggered successfully",
			"job_id", jobID,
			"run_id", run.RunID,
			"status", run.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"run": run,
	})
}

// GetRun handles GET /v1/runs/{run_id}
func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	runID := chi.URLParam(r, "run_id")

	if logger != nil {
		logger.Debug("fetching run status", "run_id", runID)
	}

	run, err := h.service.GetRun(r.Context(), runID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Debug("run status retrieved",
			"run_id", runID,
			"status", run.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"run": run,
	})
}

// StreamEvents handles GET /v1/runs/{run_id}/events
func (h *Handlers) StreamEvents(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	runID := chi.URLParam(r, "run_id")

	if logger != nil {
		logger.Info("starting event stream", "run_id", runID)
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		if logger != nil {
			logger.Error("streaming not supported by response writer")
		}
		respondError(w, r, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Send initial connection success event
	requestID := GetRequestID(r.Context())
	fmt.Fprintf(w, "event: connected\ndata: {\"request_id\":\"%s\"}\n\n", requestID)
	flusher.Flush()

	if err := h.service.StreamRunEvents(r.Context(), runID, w); err != nil {
		// Cannot change headers after streaming starts, but MUST log
		if logger != nil {
			logger.Error("streaming error occurred",
				"run_id", runID,
				"error", err,
				"error_type", fmt.Sprintf("%T", err))
		}

		// Send error event if possible (best effort)
		fmt.Fprintf(w, "event: error\ndata: {\"message\":\"stream error\",\"request_id\":\"%s\"}\n\n", requestID)
		flusher.Flush()
		return
	}

	if logger != nil {
		logger.Info("event stream completed successfully", "run_id", runID)
	}
	flusher.Flush()
}

// CancelRun handles POST /v1/runs/{run_id}/cancel
func (h *Handlers) CancelRun(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	runID := chi.URLParam(r, "run_id")

	if logger != nil {
		logger.Info("canceling run", "run_id", runID)
	}

	if err := h.service.CancelRun(r.Context(), runID); err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("run canceled successfully", "run_id", runID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondError writes a JSON error response with logging
func respondError(w http.ResponseWriter, r *http.Request, status int, message string) {
	logger := GetLogger(r.Context())
	requestID := GetRequestID(r.Context())

	// Log the error with full context
	if logger != nil {
		logger.Error("returning error response",
			"status", status,
			"message", message,
			"request_id", requestID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message":    message,
			"code":       status,
			"request_id": requestID,
		},
	})
}

// ListPipelines handles GET /v1/discovery/pipelines
func (h *Handlers) ListPipelines(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	if logger != nil {
		logger.Debug("listing pipelines from provider")
	}

	pipelines, err := h.service.ListPipelines(r.Context())
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("pipelines listed", "count", len(pipelines))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pipelines": pipelines,
	})
}

// ListPipelineJobs handles GET /v1/discovery/pipelines/{pipeline}/jobs
func (h *Handlers) ListPipelineJobs(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	pipeline := chi.URLParam(r, "pipeline")

	if logger != nil {
		logger.Debug("listing jobs from provider", "pipeline", pipeline)
	}

	jobs, err := h.service.ListPipelineJobs(r.Context(), pipeline)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("jobs listed", "pipeline", pipeline, "count", len(jobs))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}

// GetBuildDetails handles GET /v1/builds/{build_id}
func (h *Handlers) GetBuildDetails(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	buildIDStr := chi.URLParam(r, "build_id")

	buildID, err := strconv.Atoi(buildIDStr)
	if err != nil {
		if logger != nil {
			logger.Warn("invalid build_id", "build_id", buildIDStr)
		}
		respondError(w, r, http.StatusBadRequest, "invalid build_id")
		return
	}

	if logger != nil {
		logger.Debug("getting build details", "build_id", buildID)
	}

	build, plan, err := h.service.GetBuildDetails(r.Context(), buildID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("build details retrieved", "build_id", buildID, "status", build.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"build": build,
		"plan":  plan,
	})
}

// ListJobBuilds handles GET /v1/discovery/pipelines/{pipeline}/jobs/{job}/builds
func (h *Handlers) ListJobBuilds(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())
	pipeline := chi.URLParam(r, "pipeline")
	job := chi.URLParam(r, "job")

	// Parse optional limit parameter
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100 // max limit
			}
		}
	}

	if logger != nil {
		logger.Debug("listing job builds", "pipeline", pipeline, "job", job, "limit", limit)
	}

	builds, err := h.service.ListJobBuilds(r.Context(), pipeline, job, limit)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}

	if logger != nil {
		logger.Info("job builds listed", "pipeline", pipeline, "job", job, "count", len(builds))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"builds": builds,
	})
}

// handleServiceError maps service errors to HTTP responses with detailed logging
func handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	logger := GetLogger(r.Context())
	requestID := GetRequestID(r.Context())

	// Log original error with full details
	if logger != nil {
		logger.Error("service error occurred",
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
			"request_id", requestID)
	}

	switch {
	case errors.Is(err, service.ErrJobNotFound):
		respondError(w, r, http.StatusNotFound, "job not found")
	case errors.Is(err, service.ErrRunNotFound):
		respondError(w, r, http.StatusNotFound, "run not found")
	case errors.Is(err, provider.ErrJobNotFound):
		respondError(w, r, http.StatusNotFound, "job not found in provider")
	case errors.Is(err, provider.ErrRunNotFound):
		respondError(w, r, http.StatusNotFound, "run not found in provider")
	case errors.Is(err, provider.ErrUnauthorized):
		respondError(w, r, http.StatusUnauthorized, "provider authentication failed")
	case errors.Is(err, provider.ErrProviderUnavailable):
		respondError(w, r, http.StatusBadGateway, "provider temporarily unavailable")
	default:
		// Check if it's a ProviderError
		var providerErr *provider.ProviderError
		if errors.As(err, &providerErr) {
			if logger != nil {
				logger.Error("provider error details",
					"provider_code", providerErr.Code,
					"provider_message", providerErr.Message,
					"underlying_error", providerErr.Err)
			}

			if providerErr.Code >= 400 && providerErr.Code < 500 {
				respondError(w, r, providerErr.Code, providerErr.Message)
			} else {
				respondError(w, r, http.StatusBadGateway, "provider error")
			}
		} else {
			respondError(w, r, http.StatusInternalServerError, "internal server error")
		}
	}
}
