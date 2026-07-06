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
)

// NameIndex maps K8s resource names (namespace/name or bare name for cluster-scoped)
// to the UUID used in the modelsrv model. Required because modelsrv deletes by UUID
// while K8s delete events only carry the resource name.
type NameIndex struct {
	mu    sync.RWMutex
	names map[ResourceKind]map[string]uuid.UUID
}

// NewNameIndex creates an empty name index.
func NewNameIndex() *NameIndex {
	return &NameIndex{
		names: make(map[ResourceKind]map[string]uuid.UUID),
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
