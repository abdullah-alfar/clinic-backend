package ai

import "context"

// Provider is the unified AI text generation abstraction.
// Feature modules depend only on this interface, never on concrete providers.
type Provider interface {
	Generate(ctx context.Context, input string) (string, error)
}
