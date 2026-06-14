package images

import "strings"

// Search returns image names matching the query using exact, prefix, and fuzzy matching.
// For unambiguous prefix matches (exactly one result), the caller can auto-select it.
func Search(query string) []string {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	// Exact match.
	for _, name := range SupportedImages() {
		if strings.ToLower(name) == query {
			return []string{name}
		}
	}

	// Prefix match.
	var prefixMatches []string
	for _, name := range SupportedImages() {
		if strings.HasPrefix(strings.ToLower(name), query) {
			prefixMatches = append(prefixMatches, name)
		}
	}
	if len(prefixMatches) > 0 {
		return prefixMatches
	}

	// Fuzzy match — return suggestions.
	return SuggestSimilar(query, 3)
}

func SuggestSimilar(query string, maxResults int) []string {
	if maxResults <= 0 {
		maxResults = 3
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	type candidate struct {
		name string
		dist int
	}

	var candidates []candidate
	for _, name := range SupportedImages() {
		dist := levenshtein(query, strings.ToLower(name))
		if dist <= 5 {
			candidates = append(candidates, candidate{name: name, dist: dist})
		}
	}

	for _, name := range SupportedImages() {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, query) || strings.Contains(lower, query) {
			found := false
			for _, c := range candidates {
				if c.name == name {
					found = true
					break
				}
			}
			if !found {
				candidates = append(candidates, candidate{name: name, dist: 0})
			}
		}
	}

	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].dist < candidates[i].dist ||
				(candidates[j].dist == candidates[i].dist && candidates[j].name < candidates[i].name) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > maxResults {
		candidates = candidates[:maxResults]
	}

	result := make([]string, len(candidates))
	for i, c := range candidates {
		result[i] = c.name
	}
	return result
}

func levenshtein(a, b string) int {
	la := len(a)
	lb := len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
