// Package policy provides configuration-driven policy enforcement for
// DevForge operations. Policies restrict which dependencies, templates,
// and versions are permitted.
package policy

// Policy represents the organization's enforcement rules.
type Policy struct {
	// AllowedDependencies is the whitelist of permitted dependency names.
	// If empty, all dependencies are allowed.
	AllowedDependencies []string `json:"allowedDependencies" yaml:"allowed_dependencies"`

	// BlockedTemplates is the list of template URLs/names that are blocked.
	BlockedTemplates []string `json:"blockedTemplates" yaml:"blocked_templates"`

	// MaxNodeVersion is the maximum allowed Node.js major version.
	// Zero means no restriction.
	MaxNodeVersion int `json:"maxNodeVersion" yaml:"max_node_version"`

	// AllowedPlugins is the whitelist of permitted plugin names.
	// If empty, all plugins are allowed.
	AllowedPlugins []string `json:"allowedPlugins" yaml:"allowed_plugins"`

	// RequireTLS enforces TLS for all remote connections.
	RequireTLS bool `json:"requireTls" yaml:"require_tls"`
}

// IsEmpty returns true if no policy rules are defined.
func (p *Policy) IsEmpty() bool {
	return len(p.AllowedDependencies) == 0 &&
		len(p.BlockedTemplates) == 0 &&
		p.MaxNodeVersion == 0 &&
		len(p.AllowedPlugins) == 0 &&
		!p.RequireTLS
}
