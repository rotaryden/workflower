package ui_templates

import (
	_ "embed"
	htmltemplate "html/template"
	"workflower/lib/templating"
)

//go:embed base_layout.html
var baseLayoutHTML string

//go:embed start_page.html
var startPageHTML string

//go:embed review_page.html
var reviewPageHTML string

//go:embed status_page.html
var statusPageHTML string

//go:embed workflows_list.html
var workflowsListHTML string

// PageData represents the data passed to templates
type PageData struct {
	Title     string
	Workflow  any
	Workflows any
}

type TemplatesList struct {
	Start  *htmltemplate.Template
	Review *htmltemplate.Template
	Status *htmltemplate.Template
	List   *htmltemplate.Template
}

// Init initializes all templates with embedded content
func Init() (*TemplatesList, error) {
	var err error
	tplList := TemplatesList{}

	tplList.Start, err = templating.ParseHTMLTemplates("start", baseLayoutHTML, startPageHTML)
	if err != nil {
		return nil, err
	}

	tplList.Review, err = templating.ParseHTMLTemplates("review", baseLayoutHTML, reviewPageHTML)
	if err != nil {
		return nil, err
	}

	tplList.Status, err = templating.ParseHTMLTemplates("status", baseLayoutHTML, statusPageHTML)
	if err != nil {
		return nil, err
	}

	tplList.List, err = templating.ParseHTMLTemplates("list", baseLayoutHTML, workflowsListHTML)
	if err != nil {
		return nil, err
	}

	return &tplList, nil
}
