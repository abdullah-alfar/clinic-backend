package search

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// providerTimeout is the maximum time a single provider may take.
	// It is shorter than the overall handler timeout so slow providers
	// produce warnings rather than killing the whole request.
	providerTimeout = 4 * time.Second
)

// SearchService defines the contract for the global search use-case.
type SearchService interface {
	// GlobalSearch executes a concurrent search across all applicable providers
	// and returns grouped, ranked results. Partial provider failures are
	// surfaced as Warnings in SearchData rather than as a hard error.
	GlobalSearch(ctx context.Context, req SearchRequest) (SearchData, error)
}

// searchService is the production implementation of SearchService.
type searchService struct {
	registry *ProviderRegistry
}

// NewSearchService constructs a searchService backed by the given registry.
func NewSearchService(registry *ProviderRegistry) SearchService {
	return &searchService{registry: registry}
}

// providerResult is the internal message passed from a provider goroutine
// back to the collector via a buffered channel.
type providerResult struct {
	entityType EntityType
	label      string
	items      []SearchResultItem
	err        error
}

// GlobalSearch runs all enabled providers concurrently, collects results,
// ranks each group, and merges them into a single prioritised response.
//
// Concurrency model:
//  1. A buffered channel (capacity = number of providers) is created.
//  2. Each provider is launched in its own goroutine via errgroup.
//  3. Every goroutine writes exactly one providerResult to the channel.
//  4. The collector reads exactly len(providers) messages — no goroutine leaks.
//  5. Per-provider timeouts are enforced via a derived context.
//
// Partial failure:
//   - A single provider failure appends to Warnings and is logged, but does
//     NOT abort the search. The caller receives all results from healthy providers.
func (s *searchService) GlobalSearch(ctx context.Context, req SearchRequest) (SearchData, error) {
	req.Query = strings.TrimSpace(req.Query)

	// Guard: empty / too-short queries return cleanly without hitting the DB.
	if len(req.Query) < MinQueryLength {
		return SearchData{
			Query:    req.Query,
			Groups:   []SearchResultGroup{},
			Warnings: []string{},
		}, nil
	}

	// Apply default limit if the caller did not specify one.
	if req.Limit <= 0 {
		req.Limit = DefaultLimitPerProvider
	}
	if req.Limit > MaxLimitPerProvider {
		req.Limit = MaxLimitPerProvider
	}

	providers := s.registry.GetProviders(req.Types)
	if len(providers) == 0 {
		return SearchData{Query: req.Query, Groups: []SearchResultGroup{}, Warnings: []string{}}, nil
	}

	// Buffered channel: each goroutine sends exactly one result,
	// so the buffer never blocks any sender.
	resultCh := make(chan providerResult, len(providers))

	// errgroup manages goroutine lifecycle. We do NOT propagate errgroup errors
	// as hard failures; instead every goroutine always sends to resultCh so the
	// collector can read exactly len(providers) messages.
	eg, _ := errgroup.WithContext(ctx)

	for _, p := range providers {
		p := p // capture loop variable
		eg.Go(func() error {
			// Each provider gets its own deadline-bounded context derived from
			// the parent, so a slow provider cannot exceed providerTimeout.
			pCtx, cancel := context.WithTimeout(ctx, providerTimeout)
			defer cancel()

			items, err := p.Search(pCtx, req)
			resultCh <- providerResult{
				entityType: p.Type(),
				label:      p.Label(),
				items:      items,
				err:        err,
			}
			return nil // errors are surfaced via providerResult.err, not errgroup
		})
	}

	// Wait for all goroutines to complete, then close the channel.
	// We launch a companion goroutine so the collector loop below can run
	// concurrently with the Wait call.
	go func() {
		_ = eg.Wait()
		close(resultCh)
	}()

	// Collect results.
	var groups []SearchResultGroup
	var warnings []string

	for result := range resultCh {
		if result.err != nil {
			pe := &ProviderError{ProviderType: result.entityType, Cause: result.err}
			log.Printf("[search] provider error: %v", pe)
			warnings = append(warnings, pe.Error())
			continue
		}

		if len(result.items) == 0 {
			continue
		}

		// Rank and sort results for this provider group.
		ranked := RankResults(result.items, req.Query)

		groups = append(groups, SearchResultGroup{
			Type:    string(result.entityType),
			Label:   result.label,
			Count:   len(ranked),
			Results: ranked,
		})
	}

	// Sort groups by the canonical priority order defined in models.go.
	sort.Slice(groups, func(i, j int) bool {
		return priorityOf(EntityType(groups[i].Type)) < priorityOf(EntityType(groups[j].Type))
	})

	if groups == nil {
		groups = []SearchResultGroup{}
	}
	if warnings == nil {
		warnings = []string{}
	}

	return SearchData{
		Query:    req.Query,
		Groups:   groups,
		Warnings: warnings,
	}, nil
}
