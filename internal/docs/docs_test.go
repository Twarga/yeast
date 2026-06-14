package docs

import (
	"bytes"
	"strings"
	"testing"
)

func TestTopicNamesSorted(t *testing.T) {
	t.Parallel()

	names := TopicNames()
	if len(names) == 0 {
		t.Fatal("expected embedded docs topics")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("topics not sorted: %v", names)
		}
	}
}

func TestReadTopic(t *testing.T) {
	t.Parallel()

	body, err := ReadTopic("quickstart")
	if err != nil {
		t.Fatalf("ReadTopic returned error: %v", err)
	}
	if !strings.Contains(body, "# Yeast Quickstart") {
		t.Fatalf("unexpected quickstart body: %q", body)
	}
}

func TestRenderWritesMarkdownForNonTerminal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := Render(&buf, "tutorial-test"); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "# Yeast Manual Test") {
		t.Fatalf("expected raw markdown output, got %q", buf.String())
	}
}

func TestIndexMarkdown(t *testing.T) {
	t.Parallel()

	body := IndexMarkdown()
	if !strings.Contains(body, "yeast docs quickstart") {
		t.Fatalf("expected example command, got %q", body)
	}
	if !strings.Contains(body, "`tutorial-test`") {
		t.Fatalf("expected tutorial-test topic, got %q", body)
	}
	if !strings.Contains(body, "`release-smoke`") {
		t.Fatalf("expected release-smoke topic, got %q", body)
	}
}
