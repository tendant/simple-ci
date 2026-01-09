package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/lei/simple-ci/internal/config"
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

type contextKey string

const (
	contextKeyAPIKeyName contextKey = "api_key_name"
)

// Authenticate validates the API key from the Authorization header
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Expect: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "invalid authorization format, expected 'Bearer <token>'")
			return
		}

		apiKey := parts[1]
		name, valid := m.apiKeys[apiKey]
		if !valid {
			respondError(w, http.StatusUnauthorized, "invalid api key")
			return
		}

		// Add key name to context for logging/audit
		ctx := context.WithValue(r.Context(), contextKeyAPIKeyName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
