package docs

import (
	"bytes"
	"strings"
	"testing"
)

func TestTopicNamesSorted(t *testing.T) {
	t.Parallel()

	names := TopicNames()
	want := []string{"config", "installation", "quickstart", "release-smoke", "troubleshooting"}
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected topics:\n got: %v\nwant: %v", names, want)
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

func TestQuickstartNoLongerRequiresManualImagePull(t *testing.T) {
	t.Parallel()

	body, err := ReadTopic("quickstart")
	if err != nil {
		t.Fatalf("ReadTopic returned error: %v", err)
	}

	if strings.Contains(body, "yeast pull ubuntu-24.04") {
		t.Fatalf("quickstart still requires manual image pull:\n%s", body)
	}
	if !strings.Contains(body, "yeast up") {
		t.Fatalf("expected quickstart to keep yeast up flow, got %q", body)
	}
}

func TestOfflineInstallationTopicMatchesBinaryFirstInstaller(t *testing.T) {
	t.Parallel()

	body, err := ReadTopic("installation")
	if err != nil {
		t.Fatalf("ReadTopic returned error: %v", err)
	}

	if strings.Contains(body, "YEAST_REF=") {
		t.Fatalf("installation topic still uses stale YEAST_REF pinning:\n%s", body)
	}
	if strings.Contains(body, "install or bootstrap Go when needed") {
		t.Fatalf("installation topic still describes the old Go-bootstrap installer:\n%s", body)
	}
	if !strings.Contains(body, "YEAST_VERSION=") {
		t.Fatalf("expected installation topic to document YEAST_VERSION pinning, got %q", body)
	}
}

func TestReleaseSmokeUsesCurrentInstallerAndAutoPullFlow(t *testing.T) {
	t.Parallel()

	body, err := ReadTopic("release-smoke")
	if err != nil {
		t.Fatalf("ReadTopic returned error: %v", err)
	}

	if strings.Contains(body, "YEAST_REF=") {
		t.Fatalf("release-smoke topic still uses stale installer env vars:\n%s", body)
	}
	if strings.Contains(body, "yeast pull ubuntu-24.04") {
		t.Fatalf("release-smoke topic still requires manual image pull:\n%s", body)
	}
	if !strings.Contains(body, "YEAST_VERSION=") {
		t.Fatalf("expected release-smoke topic to use YEAST_VERSION pinning, got %q", body)
	}
}

func TestRenderWritesMarkdownForNonTerminal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := Render(&buf, "release-smoke"); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "# Yeast v1.1.0 Release Smoke Test") {
		t.Fatalf("expected raw markdown output, got %q", buf.String())
	}
}

func TestIndexMarkdown(t *testing.T) {
	t.Parallel()

	body := IndexMarkdown()
	if !strings.Contains(body, "yeast docs quickstart") {
		t.Fatalf("expected example command, got %q", body)
	}
	if !strings.Contains(body, "`release-smoke`") {
		t.Fatalf("expected release-smoke topic, got %q", body)
	}
	if strings.Contains(body, "`tutorial-test`") {
		t.Fatalf("did not expect obsolete tutorial-test topic, got %q", body)
	}
}
