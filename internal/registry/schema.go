// Package registry provides a client for the remote DevForge template
// registry. It supports fetching, caching, searching, and validating
// template definitions.
package registry

// Template represents a single template entry in the registry.
type Template struct {
	// Name is the unique identifier for the template.
	Name string `json:"name"`
	// Description is a human-readable summary of the template.
	Description string `json:"description"`
	// URL is the Git repository URL for cloning.
	URL string `json:"url"`
	// Tags are searchable keywords associated with the template.
	Tags []string `json:"tags"`
	// Author is the template creator.
	Author string `json:"author"`
	// Version is the template version.
	Version string `json:"version"`
}

// Registry represents the top-level registry response.
type Registry struct {
	// Templates is the list of available templates.
	Templates []Template `json:"templates"`
	// Version is the registry schema version.
	Version string `json:"version"`
}

// Validate checks that all required fields are present in a Template.
func (t Template) Validate() bool {
	return t.Name != "" && t.URL != "" && t.Description != ""
}

// ValidTemplates returns only templates that pass validation.
func (r Registry) ValidTemplates() []Template {
	var valid []Template
	for _, t := range r.Templates {
		if t.Validate() {
			valid = append(valid, t)
		}
	}
	return valid
}
