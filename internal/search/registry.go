package search

type ProviderRegistry struct {
	providers map[EntityType]SearchProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[EntityType]SearchProvider),
	}
}

func (r *ProviderRegistry) Register(p SearchProvider) {
	r.providers[p.GetEntityType()] = p
}

func (r *ProviderRegistry) GetProviders(types []string) []SearchProvider {
	if len(types) == 0 {
		var all []SearchProvider
		for _, p := range r.providers {
			all = append(all, p)
		}
		return all
	}

	var selected []SearchProvider
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	for _, p := range r.providers {
		if typeSet[string(p.GetEntityType())] {
			selected = append(selected, p)
		}
	}
	return selected
}
