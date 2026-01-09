package api

import (
	"encoding/json"
	"errors"
	"net/http"

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
	jobID := chi.URLParam(r, "job_id")

	var req struct {
		Parameters     map[string]interface{} `json:"parameters"`
		IdempotencyKey string                 `json:"idempotency_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	run, err := h.service.TriggerRun(r.Context(), jobID, req.Parameters, req.IdempotencyKey)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"run": run,
	})
}

// GetRun handles GET /v1/runs/{run_id}
func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")

	run, err := h.service.GetRun(r.Context(), runID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"run": run,
	})
}

// StreamEvents handles GET /v1/runs/{run_id}/events
func (h *Handlers) StreamEvents(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	if err := h.service.StreamRunEvents(r.Context(), runID, w); err != nil {
		// Error during streaming - client may have disconnected
		// Log but don't write response (headers already sent)
		return
	}

	flusher.Flush()
}

// CancelRun handles POST /v1/runs/{run_id}/cancel
func (h *Handlers) CancelRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")

	if err := h.service.CancelRun(r.Context(), runID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    status,
		},
	})
}

// handleServiceError maps service errors to HTTP responses
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrJobNotFound):
		respondError(w, http.StatusNotFound, "job not found")
	case errors.Is(err, service.ErrRunNotFound):
		respondError(w, http.StatusNotFound, "run not found")
	case errors.Is(err, provider.ErrJobNotFound):
		respondError(w, http.StatusNotFound, "job not found in provider")
	case errors.Is(err, provider.ErrRunNotFound):
		respondError(w, http.StatusNotFound, "run not found in provider")
	case errors.Is(err, provider.ErrUnauthorized):
		respondError(w, http.StatusUnauthorized, "provider authentication failed")
	case errors.Is(err, provider.ErrProviderUnavailable):
		respondError(w, http.StatusBadGateway, "provider temporarily unavailable")
	default:
		// Check if it's a ProviderError
		var providerErr *provider.ProviderError
		if errors.As(err, &providerErr) {
			if providerErr.Code >= 400 && providerErr.Code < 500 {
				respondError(w, providerErr.Code, providerErr.Message)
			} else {
				respondError(w, http.StatusBadGateway, "provider error")
			}
		} else {
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
	}
}
