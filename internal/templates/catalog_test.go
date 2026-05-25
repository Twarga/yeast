package templates

import (
	"reflect"
	"testing"
)

func TestBuiltinsReturnsSortedTemplates(t *testing.T) {
	t.Parallel()

	builtins, err := Builtins()
	if err != nil {
		t.Fatalf("Builtins returned error: %v", err)
	}

	got := make([]string, 0, len(builtins))
	for _, template := range builtins {
		got = append(got, template.Metadata.Name)
		if template.Source != SourceBuiltin {
			t.Fatalf("expected built-in source for %s, got %q", template.Metadata.Name, template.Source)
		}
		if template.Path == "" {
			t.Fatalf("expected built-in path for %s", template.Metadata.Name)
		}
		if len(template.Metadata.Files) == 0 {
			t.Fatalf("expected files for %s", template.Metadata.Name)
		}
	}

	want := []string{"caddy-single-vm", "two-vm-lab", "ubuntu-basic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected built-in template order:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuiltinNames(t *testing.T) {
	t.Parallel()

	got, err := BuiltinNames()
	if err != nil {
		t.Fatalf("BuiltinNames returned error: %v", err)
	}

	want := []string{"caddy-single-vm", "two-vm-lab", "ubuntu-basic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected names:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLookupBuiltin(t *testing.T) {
	t.Parallel()

	template, ok, err := LookupBuiltin("caddy-single-vm")
	if err != nil {
		t.Fatalf("LookupBuiltin returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected lookup to find caddy-single-vm")
	}
	if template.Metadata.Title != "Caddy Single VM" {
		t.Fatalf("unexpected title: %q", template.Metadata.Title)
	}
	if template.Metadata.Category != "app" {
		t.Fatalf("unexpected category: %q", template.Metadata.Category)
	}
}

func TestLookupBuiltinMissing(t *testing.T) {
	t.Parallel()

	_, ok, err := LookupBuiltin("does-not-exist")
	if err != nil {
		t.Fatalf("LookupBuiltin returned error: %v", err)
	}
	if ok {
		t.Fatal("expected missing built-in template lookup to fail")
	}
}
