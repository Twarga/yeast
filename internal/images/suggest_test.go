package images

import "testing"

func TestSuggestSimilarTypo(t *testing.T) {
	suggestions := SuggestSimilar("ubuntu-24.05", 3)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for typo")
	}
	found := false
	for _, s := range suggestions {
		if s == "ubuntu-24.04" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ubuntu-24.04 in suggestions, got %v", suggestions)
	}
}

func TestSuggestSimilarPrefix(t *testing.T) {
	suggestions := SuggestSimilar("ubuntu", 3)
	if len(suggestions) < 2 {
		t.Fatalf("expected multiple suggestions for prefix, got %v", suggestions)
	}
}

func TestSuggestSimilarNoMatch(t *testing.T) {
	suggestions := SuggestSimilar("zzzzzzzzz", 3)
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for unrelated query, got %v", suggestions)
	}
}

func TestSuggestSimilarEmpty(t *testing.T) {
	suggestions := SuggestSimilar("", 3)
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for empty query, got %v", suggestions)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "ac", 1},
		{"kitten", "sitting", 3},
		{"ubuntu-24.04", "ubuntu-24.05", 1},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSearchExactMatch(t *testing.T) {
	results := Search("ubuntu-24.04")
	if len(results) != 1 || results[0] != "ubuntu-24.04" {
		t.Fatalf("expected exact match [ubuntu-24.04], got %v", results)
	}
}

func TestSearchPrefixMatch(t *testing.T) {
	results := Search("ubuntu")
	if len(results) < 2 {
		t.Fatalf("expected multiple prefix matches for 'ubuntu', got %v", results)
	}
}

func TestSearchFuzzyMatch(t *testing.T) {
	results := Search("ubunt-24.04")
	if len(results) == 0 {
		t.Fatal("expected fuzzy suggestions for typo")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	results := Search("")
	if len(results) != 0 {
		t.Fatalf("expected no results for empty query, got %v", results)
	}
}

func TestSearchUnrelatedQuery(t *testing.T) {
	results := Search("zzzzzzzzz")
	if len(results) != 0 {
		t.Fatalf("expected no results for unrelated query, got %v", results)
	}
}
