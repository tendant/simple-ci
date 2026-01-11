package api

import (
	"context"

	"github.com/lei/simple-ci/pkg/logger"
)

// Context key constants - using plain strings for cross-package compatibility
const (
	contextKeyRequestID  = "request_id"
	contextKeyLogger     = "logger"
	contextKeyAPIKeyName = "api_key_name"
)

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return requestID
	}
	return ""
}

// GetLogger retrieves the logger from context
func GetLogger(ctx context.Context) *logger.Logger {
	if logger, ok := ctx.Value(contextKeyLogger).(*logger.Logger); ok {
		return logger
	}
	return nil
}

// GetAPIKeyName retrieves the API key name from context
func GetAPIKeyName(ctx context.Context) string {
	if name, ok := ctx.Value(contextKeyAPIKeyName).(string); ok {
		return name
	}
	return ""
}
