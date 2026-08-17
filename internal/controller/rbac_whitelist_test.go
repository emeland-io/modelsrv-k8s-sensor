package controller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWhitelisted_NilWhitelist(t *testing.T) {
	var wl *RBACWhitelist
	assert.False(t, wl.IsWhitelisted("Role", "anything"))
}

func TestIsWhitelisted_EmptyWhitelist(t *testing.T) {
	wl := &RBACWhitelist{}
	assert.False(t, wl.IsWhitelisted("Role", "admin"))
	assert.False(t, wl.IsWhitelisted("ClusterRole", "system:node"))
}

func TestIsWhitelisted_ExactMatch(t *testing.T) {
	wl := &RBACWhitelist{
		Roles:               []string{"admin"},
		ClusterRoles:        []string{"system:node"},
		RoleBindings:        []string{"default-sa-binding"},
		ClusterRoleBindings: []string{"cluster-admin"},
	}
	assert.True(t, wl.IsWhitelisted("Role", "admin"))
	assert.True(t, wl.IsWhitelisted("ClusterRole", "system:node"))
	assert.True(t, wl.IsWhitelisted("RoleBinding", "default-sa-binding"))
	assert.True(t, wl.IsWhitelisted("ClusterRoleBinding", "cluster-admin"))

	assert.False(t, wl.IsWhitelisted("Role", "other"))
	assert.False(t, wl.IsWhitelisted("ClusterRole", "other"))
}

func TestIsWhitelisted_GlobPattern(t *testing.T) {
	wl := &RBACWhitelist{
		ClusterRoles:        []string{"system:*"},
		ClusterRoleBindings: []string{"system:*"},
	}
	assert.True(t, wl.IsWhitelisted("ClusterRole", "system:node"))
	assert.True(t, wl.IsWhitelisted("ClusterRole", "system:controller:job-controller"))
	assert.True(t, wl.IsWhitelisted("ClusterRoleBinding", "system:node"))
	assert.False(t, wl.IsWhitelisted("ClusterRole", "admin"))
	assert.False(t, wl.IsWhitelisted("ClusterRole", "my-system-role"))
}

func TestIsWhitelisted_MultiplePatterns(t *testing.T) {
	wl := &RBACWhitelist{
		ClusterRoles: []string{"system:*", "kubeadm:*", "calico-*"},
	}
	assert.True(t, wl.IsWhitelisted("ClusterRole", "system:node"))
	assert.True(t, wl.IsWhitelisted("ClusterRole", "kubeadm:get-nodes"))
	assert.True(t, wl.IsWhitelisted("ClusterRole", "calico-node"))
	assert.False(t, wl.IsWhitelisted("ClusterRole", "my-app-role"))
}

func TestIsWhitelisted_UnknownKind(t *testing.T) {
	wl := &RBACWhitelist{Roles: []string{"*"}}
	assert.False(t, wl.IsWhitelisted("Unknown", "anything"))
}

func TestLoadRBACWhitelist_EmptyPath(t *testing.T) {
	wl, err := LoadRBACWhitelist("")
	require.NoError(t, err)
	assert.NotNil(t, wl)
	assert.Empty(t, wl.Roles)
}

func TestLoadRBACWhitelist_ValidFile(t *testing.T) {
	content := `
roles:
  - "local-*"
clusterRoles:
  - "system:*"
  - "kubeadm:*"
roleBindings:
  - "system:*"
clusterRoleBindings:
  - "system:*"
  - "cluster-admin"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "whitelist.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	wl, err := LoadRBACWhitelist(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"local-*"}, wl.Roles)
	assert.Equal(t, []string{"system:*", "kubeadm:*"}, wl.ClusterRoles)
	assert.Equal(t, []string{"system:*"}, wl.RoleBindings)
	assert.Equal(t, []string{"system:*", "cluster-admin"}, wl.ClusterRoleBindings)
}

func TestLoadRBACWhitelist_FileNotFound(t *testing.T) {
	_, err := LoadRBACWhitelist("/nonexistent/path.yaml")
	assert.Error(t, err)
}

func TestLoadRBACWhitelist_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":\n  :\n  [invalid"), 0644))

	_, err := LoadRBACWhitelist(path)
	assert.Error(t, err)
}

func TestLoadRBACWhitelist_InvalidPattern(t *testing.T) {
	content := `
clusterRoles:
  - "[invalid"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-pattern.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadRBACWhitelist(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pattern")
}
