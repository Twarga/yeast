package app

import (
	"reflect"
	"testing"
)

func TestListTemplatesReturnsBuiltins(t *testing.T) {
	t.Parallel()

	result, err := NewService().ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}

	got := make([]string, 0, len(result.Templates))
	for _, template := range result.Templates {
		got = append(got, template.Name)
		if template.Source != "builtin" {
			t.Fatalf("expected builtin source for %s, got %q", template.Name, template.Source)
		}
		if template.Title == "" {
			t.Fatalf("expected title for %s", template.Name)
		}
	}

	want := []string{"caddy-single-vm", "two-vm-lab", "ubuntu-basic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected template order:\n got: %#v\nwant: %#v", got, want)
	}
}
