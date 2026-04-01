package search

import "strings"

func RankResults(items []SearchResultItem, query string) {
	// Simple sorting: items with exact titles rank highest.
	// We'll let `sort.Slice` handle sorting by score.
	// Score calculation basic logic (if not already provided by SQL provider).
	
	// Pre-process query
	qLower := strings.ToLower(strings.TrimSpace(query))

	for i, item := range items {
		// If SQL didn't provide a score, or even if it did, we boost:
		score := item.Score
		
		tLower := strings.ToLower(item.Title)
		if tLower == qLower {
			score += 10.0 // exact match
		} else if strings.HasPrefix(tLower, qLower) {
			score += 5.0  // prefix match
		} else if strings.Contains(tLower, qLower) {
			score += 2.0  // substring match
		}

		sLower := strings.ToLower(item.Subtitle)
		if sLower == qLower {
			score += 8.0
		} else if strings.Contains(sLower, qLower) {
			score += 1.0
		}

		dLower := strings.ToLower(item.Description)
		if strings.Contains(dLower, qLower) {
			score += 0.5
		}

		items[i].Score = score
	}

	// Actually, let's just use `sort.Slice` to order them inline.
	// Let's implement that inside the Service, since we are returning slices.
}
