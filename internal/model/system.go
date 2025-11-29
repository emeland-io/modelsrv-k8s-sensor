package model

import (
	"github.com/google/uuid"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SystemInfo struct {
	System       *modelsrv.System
	resourceName string
	statusWriter client.SubResourceWriter
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys *modelsrv.System, name string, statusWriter client.SubResourceWriter) error {

	err := m.baseModel.AddSystem(sys)
	if err != nil {
		return err
	}

	info := &SystemInfo{
		System:       sys,
		resourceName: name,
		statusWriter: statusWriter,
	}

	m.SystemsByName[name] = info
	if sys.SystemId != uuid.Nil {
		m.SystemsByUUID[sys.SystemId] = info
	}

	return nil
}

// GetSystemById implements Model.
func (m *modelData) GetSystemById(id uuid.UUID) *SystemInfo {
	system, exists := m.SystemsByUUID[id]
	if !exists {
		return nil
	}
	return system
}

// GetSystemByResourceName implements Model.
func (m *modelData) GetSystemByResourceName(s string) *SystemInfo {
	system, exists := m.SystemsByName[s]
	if !exists {
		return nil
	}
	return system
}

// DeleteSystemById implements Model.
func (m *modelData) DeleteSystemById(id uuid.UUID) error {
	systemInfo, exists := m.SystemsByUUID[id]
	if !exists {
		return SystemNotFoundError
	}

	err := m.baseModel.DeleteSystemById(id)

	// remove system from by-id map even if error occurs, to clean up
	delete(m.SystemsByUUID, id)

	// remove system from by-name map even if error occurs, to clean up
	delete(m.SystemsByName, systemInfo.resourceName)

	return err
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemByResourceName(s string) error {
	sys, exists := m.SystemsByName[s]
	if !exists {
		return SystemNotFoundError
	}

	// remove system from by-name map even if error occurs, to clean up
	delete(m.SystemsByName, s)

	id := sys.System.SystemId
	if id != uuid.Nil {
		err := m.baseModel.DeleteSystemById(id)

		// remove system-info from by-id map even if error occurs, to clean up
		delete(m.SystemsByUUID, id)
		if err != nil {

			return err
		}
	}

	return nil
}
