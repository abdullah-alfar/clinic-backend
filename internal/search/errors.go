package search

import (
	"errors"
	"fmt"
)

// Sentinel errors for the search package.
var (
	// ErrQueryTooShort is returned when the search query is below the minimum length.
	ErrQueryTooShort = errors.New("search query is too short")

	// ErrNoProviders is returned when no providers match the requested types.
	ErrNoProviders = errors.New("no search providers available for the requested types")
)

// ProviderError wraps a per-provider failure so partial results can still be returned.
// It is NOT returned to the caller as a hard error; instead it is surfaced as a warning.
type ProviderError struct {
	ProviderType EntityType
	Cause        error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("search provider %q failed: %v", e.ProviderType, e.Cause)
}

func (e *ProviderError) Unwrap() error { return e.Cause }
