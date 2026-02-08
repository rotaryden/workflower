package templating

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

// TemplateType represents the type of template engine to use
type TemplateType int

const (
	// Text uses text/template for plain text templates (e.g., systemd service files)
	Text TemplateType = iota
	// HTML uses html/template for HTML templates with auto-escaping
	HTML
)

// Execute parses and executes a template string with the given data
// For simple one-off template execution without pre-parsing
func Execute(templateContent string, data interface{}, templateType TemplateType) (string, error) {
	var buf bytes.Buffer

	switch templateType {
	case Text:
		tmpl, err := texttemplate.New("template").Parse(templateContent)
		if err != nil {
			return "", fmt.Errorf("failed to parse text template: %w", err)
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute text template: %w", err)
		}
	case HTML:
		tmpl, err := htmltemplate.New("template").Parse(templateContent)
		if err != nil {
			return "", fmt.Errorf("failed to parse HTML template: %w", err)
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute HTML template: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported template type: %v", templateType)
	}

	return buf.String(), nil
}

// ParseText parses a text template and returns it for reuse
func ParseText(name, content string) (*texttemplate.Template, error) {
	tmpl, err := texttemplate.New(name).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text template %s: %w", name, err)
	}
	return tmpl, nil
}

// ParseHTMLTemplates parses multiple HTML template strings into a single template set
// The first template is the main template, additional templates are parsed into it
func ParseHTMLTemplates(name string, templates ...string) (*htmltemplate.Template, error) {
	if len(templates) == 0 {
		return nil, fmt.Errorf("at least one template is required")
	}

	tmpl, err := htmltemplate.New(name).Parse(templates[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse base layout template %s: %w", name, err)
	}

	for i := 1; i < len(templates); i++ {
		tmpl, err = tmpl.Parse(templates[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse additional template %d: %w", i, err)
		}
	}

	return tmpl, nil
}

// ExecuteToWriter executes a text template directly to a writer
func ExecuteToWriter(w interface{ Write([]byte) (int, error) }, tmpl *texttemplate.Template, data interface{}) error {
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

// ExecuteHTMLToWriter executes an HTML template directly to a writer
func ExecuteHTMLToWriter(w interface{ Write([]byte) (int, error) }, tmpl *htmltemplate.Template, data interface{}) error {
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
