package search

import (
	"sort"
	"strings"
	"time"
)

// RankResults enriches the Score field of each SearchResultItem based on
// how well it matches the query string, then returns the slice sorted
// descending by score. The original slice is mutated in place and also
// returned for convenience.
//
// Scoring tiers (additive):
//
//	+10.0  exact title match
//	+5.0   title prefix match
//	+2.0   title substring match
//	+8.0   exact subtitle match (phone / email exact hit)
//	+1.0   subtitle substring match
//	+0.5   description substring match
//	+1.0   recency bonus (created_at within last 30 days, if present in metadata)
func RankResults(items []SearchResultItem, query string) []SearchResultItem {
	qLower := strings.ToLower(strings.TrimSpace(query))

	for i, item := range items {
		score := item.Score // preserve any score pre-set by the DB (e.g. FTS rank)

		tLower := strings.ToLower(item.Title)
		switch {
		case tLower == qLower:
			score += 10.0 // exact match
		case strings.HasPrefix(tLower, qLower):
			score += 5.0 // prefix match
		case strings.Contains(tLower, qLower):
			score += 2.0 // substring match
		}

		sLower := strings.ToLower(item.Subtitle)
		switch {
		case sLower == qLower:
			score += 8.0 // exact subtitle/phone/email hit
		case strings.Contains(sLower, qLower):
			score += 1.0
		}

		if strings.Contains(strings.ToLower(item.Description), qLower) {
			score += 0.5
		}

		// Recency bonus: +1.0 if created_at metadata field is within the last 30 days.
		if raw, ok := item.Metadata["created_at"]; ok {
			if t, ok := raw.(time.Time); ok {
				if time.Since(t) <= 30*24*time.Hour {
					score += 1.0
				}
			}
		}

		items[i].Score = score
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	return items
}
