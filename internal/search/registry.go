package search

// ProviderRegistry maintains the set of registered SearchProviders.
// Providers are keyed by their EntityType to prevent duplicates.
// This registry is constructed once at application startup and is read-only
// after registration — safe for concurrent reads without locking.
type ProviderRegistry struct {
	providers map[EntityType]SearchProvider
}

// NewProviderRegistry returns an empty registry ready for registration.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[EntityType]SearchProvider),
	}
}

// Register adds a provider to the registry.
// If a provider for the same EntityType already exists it is silently replaced.
// Register must only be called during application initialisation (not concurrently).
func (r *ProviderRegistry) Register(p SearchProvider) {
	r.providers[p.Type()] = p
}

// GetProviders returns the slice of providers to invoke for a given request.
//   - If types is empty, all registered providers are returned.
//   - Otherwise only the providers whose EntityType appears in types are returned.
func (r *ProviderRegistry) GetProviders(types []string) []SearchProvider {
	if len(types) == 0 {
		all := make([]SearchProvider, 0, len(r.providers))
		for _, p := range r.providers {
			all = append(all, p)
		}
		return all
	}

	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	selected := make([]SearchProvider, 0, len(types))
	for _, p := range r.providers {
		if _, ok := typeSet[string(p.Type())]; ok {
			selected = append(selected, p)
		}
	}
	return selected
}
