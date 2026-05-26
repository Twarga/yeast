package app

import "yeast/internal/templates"

type TemplateSummary struct {
	Name        string `json:"Name"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Category    string `json:"Category"`
	Version     string `json:"Version"`
	Source      string `json:"Source"`
	Path        string `json:"Path,omitempty"`
}

type TemplateListResult struct {
	Templates []TemplateSummary `json:"Templates"`
}

func (s *Service) ListTemplates() (TemplateListResult, error) {
	builtins, err := templates.Builtins()
	if err != nil {
		return TemplateListResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	result := TemplateListResult{Templates: make([]TemplateSummary, 0, len(builtins))}
	for _, template := range builtins {
		result.Templates = append(result.Templates, templateSummary(template))
	}
	return result, nil
}

func templateSummary(template templates.Template) TemplateSummary {
	return TemplateSummary{
		Name:        template.Metadata.Name,
		Title:       template.Metadata.Title,
		Description: template.Metadata.Description,
		Category:    template.Metadata.Category,
		Version:     template.Metadata.Version,
		Source:      string(template.Source),
		Path:        template.Path,
	}
}
