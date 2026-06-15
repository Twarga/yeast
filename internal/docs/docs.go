package docs

import (
	"embed"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

//go:embed embedded/*.md
var content embed.FS

var topics = map[string]string{
	"config":          "embedded/config.md",
	"installation":    "embedded/installation.md",
	"quickstart":      "embedded/quickstart.md",
	"release-smoke":   "embedded/release-smoke.md",
	"troubleshooting": "embedded/troubleshooting.md",
}

func TopicNames() []string {
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func HasTopic(name string) bool {
	_, ok := topics[name]
	return ok
}

func ReadTopic(name string) (string, error) {
	path, ok := topics[name]
	if !ok {
		return "", fmt.Errorf("unknown docs topic %q", name)
	}
	body, err := content.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read topic %q: %w", name, err)
	}
	return string(body), nil
}

func Render(w io.Writer, name string) error {
	body, err := ReadTopic(name)
	if err != nil {
		return err
	}

	if file, ok := w.(interface{ Fd() uintptr }); ok && term.IsTerminal(int(file.Fd())) {
		width := 80
		if terminalWidth, _, sizeErr := term.GetSize(int(file.Fd())); sizeErr == nil && terminalWidth > 0 {
			if terminalWidth > 120 {
				width = 120
			} else {
				width = terminalWidth
			}
		}
		renderer, renderErr := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if renderErr != nil {
			return fmt.Errorf("create docs renderer: %w", renderErr)
		}
		rendered, renderErr := renderer.Render(body)
		if renderErr != nil {
			return fmt.Errorf("render topic %q: %w", name, renderErr)
		}
		_, err = io.WriteString(w, rendered)
		return err
	}

	_, err = io.WriteString(w, body)
	return err
}

func DefaultTopic() string {
	return "quickstart"
}

func IndexMarkdown() string {
	var b strings.Builder
	b.WriteString("# Yeast Terminal Docs\n\n")
	b.WriteString("Available topics:\n\n")
	for _, name := range TopicNames() {
		b.WriteString("- `" + name + "`\n")
	}
	b.WriteString("\nExamples:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("yeast docs quickstart\n")
	b.WriteString("yeast docs installation\n")
	b.WriteString("yeast docs release-smoke\n")
	b.WriteString("yeast docs --list\n")
	b.WriteString("```\n")
	return b.String()
}
