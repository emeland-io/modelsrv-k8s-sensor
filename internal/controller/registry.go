package controller

import (
	"sync"

	"github.com/google/uuid"
)

// ResourceKind identifies a K8s-derived resource type in the name index.
type ResourceKind string

const (
	KindContext           ResourceKind = "Context"
	KindSystem            ResourceKind = "System"
	KindAPI               ResourceKind = "API"
	KindComponent         ResourceKind = "Component"
	KindSystemInstance    ResourceKind = "SystemInstance"
	KindAPIInstance       ResourceKind = "APIInstance"
	KindComponentInstance ResourceKind = "ComponentInstance"
	KindRole              ResourceKind = "Role"
	KindBinding           ResourceKind = "Binding"
	KindArtifact          ResourceKind = "Artifact"
	KindArtifactInstance  ResourceKind = "ArtifactInstance"
)

// NameIndex maps K8s resource names (namespace/name or bare name for cluster-scoped)
// to the UUID used in the modelsrv model. Required because modelsrv deletes by UUID
// while K8s delete events only carry the resource name.
//
// It also maintains a reverse Helm ownership index: for each resource deployed by
// a Helm release, it stores a mapping from (ResourceKind, namespace/name) to the
// SystemInstance UUID of the owning release. This enables workload/API controllers
// to set the SystemInstance ref regardless of reconciliation ordering.
//
// Additionally it tracks pending role-binding references: when a RoleBinding
// reconciles before its referenced Role, it records itself so the Role controller
// can trigger re-reconciliation once the Role appears.
type NameIndex struct {
	mu    sync.RWMutex
	names map[ResourceKind]map[string]uuid.UUID

	// helmOwner maps ResourceKind -> "namespace/name" -> SystemInstance UUID.
	// Populated by the HelmRelease controller, read by workload/API controllers.
	helmOwner map[ResourceKind]map[string]uuid.UUID

	// pendingBindings maps roleIndexKey -> set of binding NamespacedName strings
	// that are waiting for the role to appear in the index.
	pendingBindings map[string]map[string]struct{}
}

// NewNameIndex creates an empty name index.
func NewNameIndex() *NameIndex {
	return &NameIndex{
		names:           make(map[ResourceKind]map[string]uuid.UUID),
		helmOwner:       make(map[ResourceKind]map[string]uuid.UUID),
		pendingBindings: make(map[string]map[string]struct{}),
	}
}

// Put records a name -> UUID mapping for the given resource kind.
func (idx *NameIndex) Put(kind ResourceKind, name string, id uuid.UUID) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	m, ok := idx.names[kind]
	if !ok {
		m = make(map[string]uuid.UUID)
		idx.names[kind] = m
	}
	m[name] = id
}

// Get returns the UUID for a resource name, or uuid.Nil if not found.
func (idx *NameIndex) Get(kind ResourceKind, name string) uuid.UUID {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	m, ok := idx.names[kind]
	if !ok {
		return uuid.Nil
	}
	return m[name]
}

// Delete removes a name mapping and returns the UUID that was stored, or uuid.Nil.
func (idx *NameIndex) Delete(kind ResourceKind, name string) uuid.UUID {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	m, ok := idx.names[kind]
	if !ok {
		return uuid.Nil
	}
	id, ok := m[name]
	if !ok {
		return uuid.Nil
	}
	delete(m, name)
	return id
}

// Keys returns a snapshot of all name keys currently indexed for the given kind.
func (idx *NameIndex) Keys(kind ResourceKind) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	m, ok := idx.names[kind]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SetHelmOwner records that a resource (identified by kind + namespace/name) is
// owned by the given SystemInstance (Helm release).
func (idx *NameIndex) SetHelmOwner(kind ResourceKind, name string, systemInstanceID uuid.UUID) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	m, ok := idx.helmOwner[kind]
	if !ok {
		m = make(map[string]uuid.UUID)
		idx.helmOwner[kind] = m
	}
	m[name] = systemInstanceID
}

// GetHelmOwner returns the SystemInstance UUID that owns the given resource,
// or uuid.Nil if the resource is not part of any tracked Helm release.
func (idx *NameIndex) GetHelmOwner(kind ResourceKind, name string) uuid.UUID {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	m, ok := idx.helmOwner[kind]
	if !ok {
		return uuid.Nil
	}
	return m[name]
}

// DeleteHelmOwner removes a single resource ownership entry.
func (idx *NameIndex) DeleteHelmOwner(kind ResourceKind, name string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if m, ok := idx.helmOwner[kind]; ok {
		delete(m, name)
	}
}

// DeleteHelmOwnersBySystemInstance removes all ownership entries pointing to
// the given SystemInstance UUID. Used when a Helm release is deleted.
func (idx *NameIndex) DeleteHelmOwnersBySystemInstance(systemInstanceID uuid.UUID) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, m := range idx.helmOwner {
		for k, v := range m {
			if v == systemInstanceID {
				delete(m, k)
			}
		}
	}
}

// AddPendingBinding records that a RoleBinding (identified by its NamespacedName
// string) is waiting for the given roleIndexKey to appear in the index.
func (idx *NameIndex) AddPendingBinding(roleIndexKey string, bindingName string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	m, ok := idx.pendingBindings[roleIndexKey]
	if !ok {
		m = make(map[string]struct{})
		idx.pendingBindings[roleIndexKey] = m
	}
	m[bindingName] = struct{}{}
}

// RemovePendingBinding removes a single pending-binding entry, e.g. when the
// binding is deleted before the role arrives.
func (idx *NameIndex) RemovePendingBinding(roleIndexKey string, bindingName string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if m, ok := idx.pendingBindings[roleIndexKey]; ok {
		delete(m, bindingName)
		if len(m) == 0 {
			delete(idx.pendingBindings, roleIndexKey)
		}
	}
}

// RemovePendingBindingByName removes a binding from all pending sets regardless
// of which role it was waiting on. Used when a binding is deleted and we no
// longer know its roleRef.
func (idx *NameIndex) RemovePendingBindingByName(bindingName string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for roleKey, m := range idx.pendingBindings {
		delete(m, bindingName)
		if len(m) == 0 {
			delete(idx.pendingBindings, roleKey)
		}
	}
}

// ResolvePendingBindings drains the pending set for the given roleIndexKey and
// returns the set of binding NamespacedName strings that were waiting. The
// caller is responsible for re-enqueuing them into the appropriate binding
// reconciler(s).
func (idx *NameIndex) ResolvePendingBindings(roleIndexKey string) []string {
	idx.mu.Lock()
	pending, ok := idx.pendingBindings[roleIndexKey]
	if ok {
		delete(idx.pendingBindings, roleIndexKey)
	}
	idx.mu.Unlock()

	if !ok {
		return nil
	}
	names := make([]string, 0, len(pending))
	for name := range pending {
		names = append(names, name)
	}
	return names
}
