package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/lei/simple-ci/internal/config"
	"github.com/lei/simple-ci/pkg/logger"
)

// AuthMiddleware handles API key authentication
type AuthMiddleware struct {
	apiKeys map[string]string // key -> name
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(keys []config.APIKey) *AuthMiddleware {
	keyMap := make(map[string]string)
	for _, k := range keys {
		keyMap[k.Key] = k.Name
	}
	return &AuthMiddleware{apiKeys: keyMap}
}

// Authenticate validates the API key from the Authorization header
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := GetLogger(r.Context())

		if logger != nil {
			logger.Debug("authenticating request")
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			if logger != nil {
				logger.Warn("authentication failed: missing authorization header")
			}
			respondError(w, r, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Expect: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			if logger != nil {
				logger.Warn("authentication failed: invalid authorization format")
			}
			respondError(w, r, http.StatusUnauthorized, "invalid authorization format, expected 'Bearer <token>'")
			return
		}

		apiKey := parts[1]
		name, valid := m.apiKeys[apiKey]
		if !valid {
			if logger != nil {
				keyPrefix := apiKey
				if len(apiKey) > 8 {
					keyPrefix = apiKey[:8]
				}
				logger.Warn("authentication failed: invalid api key", "key_prefix", keyPrefix)
			}
			respondError(w, r, http.StatusUnauthorized, "invalid api key")
			return
		}

		if logger != nil {
			logger.Debug("authentication successful", "api_key_name", name)
		}

		// Add key name to context for logging/audit
		ctx := context.WithValue(r.Context(), contextKeyAPIKeyName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware adds structured logging to all requests
type LoggingMiddleware struct {
	logger *logger.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

// Handler wraps HTTP handlers with logging
func (m *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get request ID from chi's middleware
		requestID := middleware.GetReqID(r.Context())
		if requestID == "" {
			requestID = "unknown"
		}

		// Create request-scoped logger
		reqLogger := m.logger.With(
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		// Add logger and request ID to context
		ctx := context.WithValue(r.Context(), contextKeyLogger, reqLogger)
		ctx = context.WithValue(ctx, contextKeyRequestID, requestID)

		// Wrap response writer to capture status and bytes
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Log request start
		reqLogger.Debug("request started",
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())

		start := time.Now()
		defer func() {
			duration := time.Since(start)
			
			if wrapped.statusCode >= 500 {
				reqLogger.Error("request completed",
					"status", wrapped.statusCode,
					"duration_ms", duration.Milliseconds(),
					"bytes_written", wrapped.bytesWritten)
			} else if wrapped.statusCode >= 400 {
				reqLogger.Warn("request completed",
					"status", wrapped.statusCode,
					"duration_ms", duration.Milliseconds(),
					"bytes_written", wrapped.bytesWritten)
			} else {
				reqLogger.Info("request completed",
					"status", wrapped.statusCode,
					"duration_ms", duration.Milliseconds(),
					"bytes_written", wrapped.bytesWritten)
			}
		}()

		next.ServeHTTP(wrapped, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
