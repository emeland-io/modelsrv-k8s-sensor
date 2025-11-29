package model

import (
	"github.com/google/uuid"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ComponentInfo struct {
	Component    *modelsrv.Component
	resourceName string
	statusWriter client.SubResourceWriter
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp *modelsrv.Component, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddComponent(comp)
	if err != nil {
		return err
	}

	info := &ComponentInfo{
		Component:    comp,
		resourceName: name,
		statusWriter: writer,
	}

	m.ComponentsByName[name] = info
	if comp.ComponentId != uuid.Nil {
		m.ComponentsByUUID[comp.ComponentId] = info
	}
	return nil
}

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) *ComponentInfo {
	comp, exists := m.ComponentsByUUID[id]
	if !exists {
		return nil
	}
	return comp
}

// GetComponentByResourceName implements Model.
func (m *modelData) GetComponentByResourceName(s string) *ComponentInfo {
	comp, exists := m.ComponentsByName[s]
	if !exists {
		return nil
	}
	return comp
}

// DeleteComponentById implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	compInfo, exists := m.ComponentsByUUID[id]
	if !exists {
		return ComponentNotFoundError
	}

	err := m.baseModel.DeleteComponentById(id)

	// remove component from by-id map even if error occurs, to clean up
	delete(m.ComponentsByUUID, id)

	// remove component from by-name map even if error occurs, to clean up
	delete(m.ComponentsByName, compInfo.resourceName)

	return err
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentByResourceName(s string) error {
	compInfo, exists := m.ComponentsByName[s]
	if !exists {
		return ComponentNotFoundError
	}

	err := m.baseModel.DeleteComponentById(compInfo.Component.ComponentId)

	delete(m.ComponentsByName, s)
	if compInfo.Component.ComponentId != uuid.Nil {
		delete(m.ComponentsByUUID, compInfo.Component.ComponentId)
	}
	return err
}
