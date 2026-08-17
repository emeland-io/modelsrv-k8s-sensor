package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RBACWhitelist defines name patterns for RBAC resources that should not
// generate findings when missing annotation references. This suppresses
// noise from system-managed K8s RBAC resources that will never be annotated
// with EmELand references.
type RBACWhitelist struct {
	// Roles lists glob patterns matching Role names to suppress.
	Roles []string `yaml:"roles,omitempty"`
	// ClusterRoles lists glob patterns matching ClusterRole names to suppress.
	ClusterRoles []string `yaml:"clusterRoles,omitempty"`
	// RoleBindings lists glob patterns matching RoleBinding names to suppress.
	RoleBindings []string `yaml:"roleBindings,omitempty"`
	// ClusterRoleBindings lists glob patterns matching ClusterRoleBinding names to suppress.
	ClusterRoleBindings []string `yaml:"clusterRoleBindings,omitempty"`
}

// LoadRBACWhitelist reads and parses a YAML whitelist file. If path is empty,
// returns an empty (no-op) whitelist.
func LoadRBACWhitelist(path string) (*RBACWhitelist, error) {
	if path == "" {
		return &RBACWhitelist{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wl RBACWhitelist
	if err := yaml.Unmarshal(data, &wl); err != nil {
		return nil, err
	}
	if err := wl.validate(); err != nil {
		return nil, err
	}
	return &wl, nil
}

// IsWhitelisted returns true if the given resource name matches any pattern
// in the list for the given kind. The kind parameter should be one of:
// "Role", "ClusterRole", "RoleBinding", "ClusterRoleBinding".
func (wl *RBACWhitelist) IsWhitelisted(kind, name string) bool {
	if wl == nil {
		return false
	}
	var patterns []string
	switch kind {
	case "Role":
		patterns = wl.Roles
	case "ClusterRole":
		patterns = wl.ClusterRoles
	case "RoleBinding":
		patterns = wl.RoleBindings
	case "ClusterRoleBinding":
		patterns = wl.ClusterRoleBindings
	default:
		return false
	}
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// validate checks that all patterns are syntactically valid globs.
func (wl *RBACWhitelist) validate() error {
	all := []struct {
		field    string
		patterns []string
	}{
		{"roles", wl.Roles},
		{"clusterRoles", wl.ClusterRoles},
		{"roleBindings", wl.RoleBindings},
		{"clusterRoleBindings", wl.ClusterRoleBindings},
	}
	for _, f := range all {
		for _, p := range f.patterns {
			if _, err := filepath.Match(p, ""); err != nil {
				return fmt.Errorf("%s: invalid pattern %q: %w", f.field, p, err)
			}
		}
	}
	return nil
}
