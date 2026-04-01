package search

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type SearchService interface {
	GlobalSearch(ctx context.Context, tenantID uuid.UUID, query string, types []string) (SearchData, error)
}

type searchService struct {
	registry *ProviderRegistry
}

func NewSearchService(registry *ProviderRegistry) SearchService {
	return &searchService{registry: registry}
}

func (s *searchService) GlobalSearch(ctx context.Context, tenantID uuid.UUID, query string, types []string) (SearchData, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return SearchData{
			Query:  query,
			Groups: []SearchResultGroup{},
		}, nil
	}

	providers := s.registry.GetProviders(types)
	if len(providers) == 0 {
		return SearchData{Query: query, Groups: []SearchResultGroup{}}, nil
	}

	limitPerType := 20

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allGroups []SearchResultGroup
	var searchErr error

	// Run all registered providers concurrently
	for _, p := range providers {
		wg.Add(1)
		go func(provider SearchProvider) {
			defer wg.Done()
			
			// We can pass a shorter timeout ctx for each provider if needed,
			// but we rely on the parent ctx timeout for now.
			results, err := provider.Search(ctx, tenantID, query, limitPerType)
			
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				// We log or capture error, but ideally one failing provider shouldn't
				// kill the whole request, but for strictness we return the first error.
				searchErr = err
				return
			}
			
			// Only append non-empty groups or optionally empty groups for UX completeness
			if len(results) > 0 {
			    RankResults(results, query)
                // Sort by score
				sort.Slice(results, func(i, j int) bool {
                    return results[i].Score > results[j].Score
                })
				
				allGroups = append(allGroups, SearchResultGroup{
					Type:    string(provider.GetEntityType()),
					Label:   provider.GetEntityLabel(),
					Count:   len(results),
					Results: results,
				})
			}
		}(p)
	}

	wg.Wait()

	if searchErr != nil {
		return SearchData{}, searchErr
	}

	// Optionally sort groups (e.g. by Type name or total score)
	sort.Slice(allGroups, func(i, j int) bool {
		return allGroups[i].Type < allGroups[j].Type
	})

	if allGroups == nil {
		allGroups = []SearchResultGroup{}
	}

	return SearchData{
		Query:  query,
		Groups: allGroups,
	}, nil
}
