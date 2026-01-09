package provider

import (
	"errors"
	"fmt"
)

var (
	// ErrJobNotFound indicates the job doesn't exist in the provider
	ErrJobNotFound = errors.New("job not found in provider")

	// ErrRunNotFound indicates the run doesn't exist in the provider
	ErrRunNotFound = errors.New("run not found in provider")

	// ErrUnauthorized indicates provider authentication failed
	ErrUnauthorized = errors.New("provider authentication failed")

	// ErrProviderUnavailable indicates the provider is temporarily unavailable
	ErrProviderUnavailable = errors.New("provider temporarily unavailable")
)

// ProviderError represents a provider-specific error
type ProviderError struct {
	Code    int
	Message string
	Err     error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("provider error %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("provider error %d: %s", e.Code, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
