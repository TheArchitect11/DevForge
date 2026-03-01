package policy

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/semver"
)

// Engine enforces policies against proposed operations.
type Engine struct {
	policy *Policy
	log    *logger.Logger
}

// NewEngine creates a policy engine with the given rules.
func NewEngine(p *Policy, log *logger.Logger) *Engine {
	return &Engine{policy: p, log: log}
}

// Violation represents a single policy violation.
type Violation struct {
	Rule    string
	Message string
}

// CheckDependency validates that a dependency is allowed by policy.
func (e *Engine) CheckDependency(name, version string) *Violation {
	if e.policy == nil || e.policy.IsEmpty() {
		return nil
	}

	// Check allowed dependencies whitelist.
	if len(e.policy.AllowedDependencies) > 0 {
		allowed := false
		for _, dep := range e.policy.AllowedDependencies {
			if strings.EqualFold(dep, name) {
				allowed = true
				break
			}
		}
		if !allowed {
			v := &Violation{
				Rule:    "allowed_dependencies",
				Message: fmt.Sprintf("dependency %q is not in the allowed list", name),
			}
			e.log.Warn(fmt.Sprintf("policy violation: %s", v.Message))
			return v
		}
	}

	// Check max node version.
	if e.policy.MaxNodeVersion > 0 && strings.EqualFold(name, "node") && version != "" && version != "latest" {
		parsed, err := semver.Parse(version)
		if err == nil && parsed.Major > e.policy.MaxNodeVersion {
			v := &Violation{
				Rule:    "max_node_version",
				Message: fmt.Sprintf("node version %q exceeds maximum allowed version %d", version, e.policy.MaxNodeVersion),
			}
			e.log.Warn(fmt.Sprintf("policy violation: %s", v.Message))
			return v
		}
	}

	return nil
}

// CheckTemplate validates that a template URL is not blocked.
func (e *Engine) CheckTemplate(templateURL string) *Violation {
	if e.policy == nil || e.policy.IsEmpty() {
		return nil
	}

	for _, blocked := range e.policy.BlockedTemplates {
		if strings.Contains(strings.ToLower(templateURL), strings.ToLower(blocked)) {
			v := &Violation{
				Rule:    "blocked_templates",
				Message: fmt.Sprintf("template %q is blocked by policy", templateURL),
			}
			e.log.Warn(fmt.Sprintf("policy violation: %s", v.Message))
			return v
		}
	}

	return nil
}

// CheckPlugin validates that a plugin is allowed to run.
func (e *Engine) CheckPlugin(pluginName string) *Violation {
	if e.policy == nil || e.policy.IsEmpty() {
		return nil
	}

	if len(e.policy.AllowedPlugins) == 0 {
		return nil
	}

	for _, name := range e.policy.AllowedPlugins {
		if strings.EqualFold(name, pluginName) {
			return nil
		}
	}

	v := &Violation{
		Rule:    "allowed_plugins",
		Message: fmt.Sprintf("plugin %q is not in the allowed list", pluginName),
	}
	e.log.Warn(fmt.Sprintf("policy violation: %s", v.Message))
	return v
}

// ValidateAll runs all policy checks for a provisioning request and
// returns a combined list of violations.
func (e *Engine) ValidateAll(deps []string, versions map[string]string, templateURL string) []Violation {
	var violations []Violation

	for _, dep := range deps {
		ver := versions[dep]
		if v := e.CheckDependency(dep, ver); v != nil {
			violations = append(violations, *v)
		}
	}

	if v := e.CheckTemplate(templateURL); v != nil {
		violations = append(violations, *v)
	}

	return violations
}
