package project

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID returned error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty project id")
	}
	if !strings.HasPrefix(id, ProjectIDPrefix) {
		t.Fatalf("expected project id prefix %q, got %q", ProjectIDPrefix, id)
	}
	if !IsValidID(id) {
		t.Fatalf("expected path-safe project id, got %q", id)
	}
}

func TestGenerateIDDifferentAcrossCalls(t *testing.T) {
	first, err := GenerateID()
	if err != nil {
		t.Fatalf("first GenerateID returned error: %v", err)
	}
	second, err := GenerateID()
	if err != nil {
		t.Fatalf("second GenerateID returned error: %v", err)
	}
	if first == second {
		t.Fatalf("expected different ids, got %q twice", first)
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid", id: "proj_0123456789abcdef01234567", want: true},
		{name: "missing prefix", id: "0123456789abcdef01234567", want: false},
		{name: "uppercase", id: "proj_0123456789ABCDEF01234567", want: false},
		{name: "too short", id: "proj_012345", want: false},
		{name: "path separator", id: "proj_0123456789abcdef0123/567", want: false},
		{name: "path traversal", id: "../proj_0123456789abcdef01234567", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidID(tc.id); got != tc.want {
				t.Fatalf("IsValidID(%q)=%v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
